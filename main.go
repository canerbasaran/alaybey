package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

var folderToWatch, indexPage string
var port int
var messageChannels = make(map[chan []byte]bool)
var quit = make(chan bool)
var isRoutineOpen = false

const jsFile = "35b96a650c2fbb93d23dd81116b561b8.js"

func main() {
	flag.IntVar(&port, "p", 8003, "port to serve")
	flag.StringVar(&indexPage, "i", "index.html", "index page to render on /")
	flag.StringVar(&folderToWatch, "f", ".", "folder to watch (default: current)")
	flag.Parse()
	var err error
	err = serve()
	if err != nil {
		log.Println(err)
	}
}

func serve() (err error) {
	go watchFileSystem()
	fmt.Println("listening on :", port)
	http.HandleFunc("/", handler)
	http.HandleFunc("/sse", handleSSE)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/"+jsFile {
		w.Write([]byte(js))
		return
	}

	if r.URL.Path == "/" {
		r.URL.Path = "/" + indexPage
	}
	ext := filepath.Ext(r.URL.Path)

	var b []byte
	b, err := ioutil.ReadFile(path.Join(folderToWatch, path.Clean(r.URL.Path[1:])))
	if err != nil {
		log.Println("could not find file")
		return
	}

	if ext == ".css" {
		w.Header().Set("Content-Type", "text/css")
	}

	if ext == ".html" {
		b = bytes.Replace(b,
			[]byte("</body>"),
			[]byte(fmt.Sprintf(`<script src="/%s"></script></body>`, jsFile)),
			1,
		)
	}

	w.Write(b)
}

func watchFileSystem() (err error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	lastEvent := time.Now()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				_, fname := filepath.Split(event.Name)
				if time.Since(lastEvent).Nanoseconds() > (50*time.Millisecond).Nanoseconds() && !strings.HasPrefix(fname, ".") && !strings.HasSuffix(fname, "~") {
					lastEvent = time.Now()
					go func() {
						for messageChannel := range messageChannels {
							messageChannel <- []byte("")
						}
					}()
				}
			}
		}
	}()

	filepath.Walk(folderToWatch, func(path string, fi os.FileInfo, err error) error {
		if fi.Mode().IsDir() {
			return watcher.Add(path)
		}
		return nil
	})

	<-done
	return
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	_messageChannel := make(chan []byte)
	messageChannels[_messageChannel] = true
	if isRoutineOpen {
		quit <- true
		isRoutineOpen = false
	}

	for {
		select {
		case <-_messageChannel:
			w.Write([]byte("data: change\n\n"))
			w.(http.Flusher).Flush()
		case <-r.Context().Done():
			delete(messageChannels, _messageChannel)
			if len(messageChannels) == 0 {
				go checkIdle()
				isRoutineOpen = true
			}
			return
		}
	}
}

func checkIdle() {
	select {
	case <-quit:
		return
	case <-time.Tick(time.Second * 3):
		os.Exit(3)
	}
}

const js = `// scroll saving
var cookieName = "page_scroll";
var expdays = 365;

// An adaptation of Dorcht's cookie functions.

function setCookie(name, value, expires, path, domain, secure) {
    if (!expires) expires = new Date();
    document.cookie = name + "=" + escape(value) + 
        ((expires == null) ? "" : "; expires=" + expires.toGMTString()) +
        ((path    == null) ? "" : "; path=" + path) +
        ((domain  == null) ? "" : "; domain=" + domain) +
        ((secure  == null) ? "" : "; secure");
}

function getCookie(name) {
    var arg = name + "=";
    var alen = arg.length;
    var clen = document.cookie.length;
    var i = 0;
    while (i < clen) {
        var j = i + alen;
        if (document.cookie.substring(i, j) == arg) {
            return getCookieVal(j);
        }
        i = document.cookie.indexOf(" ", i) + 1;
        if (i == 0) break;
    }
    return null;
}

function getCookieVal(offset) {
    var endstr = document.cookie.indexOf(";", offset);
    if (endstr == -1) endstr = document.cookie.length;
    return unescape(document.cookie.substring(offset, endstr));
}

function deleteCookie(name, path, domain) {
    document.cookie = name + "=" +
        ((path   == null) ? "" : "; path=" + path) +
        ((domain == null) ? "" : "; domain=" + domain) +
        "; expires=Thu, 01-Jan-00 00:00:01 GMT";
}

function saveScroll() {
    var expdate = new Date();
    expdate.setTime(expdate.getTime() + (expdays*24*60*60*1000)); // expiry date

    var x = document.pageXOffset || document.body.scrollLeft || window.scrollX;
    var y = document.pageYOffset || document.body.scrollTop || window.scrollY;
    var data = x + "_" + y;
    setCookie(cookieName, data, expdate);
}

function loadScroll() {
    var inf = getCookie(cookieName);
    if (!inf) { return; }
    var ar = inf.split("_");
    if (ar.length == 2) {
        window.scrollTo(parseInt(ar[0]), parseInt(ar[1]));
    }
}

document.addEventListener("DOMContentLoaded", function() {
    loadScroll();
});

window.addEventListener("beforeunload", function (event) {
    saveScroll();
});


// sse
var eventListener = new EventSource(window.origin + '/sse')
eventListener.onmessage = (event) => {
    location.reload();
}
`
