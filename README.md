# alaybey

It's a simple stupid live-reloading tool for web development.

## Install for non-Go users

```sh
curl -sf https://gobinaries.com/canerbasaran/alaybey | sh
```

## Install

```
go get github.com/canerbasaran/alaybey
```

## Usage 

### Basic usage

You can just run `alaybey` in a directory:

```bash
$ alaybey
```

And then load your browser to `localhost:8003` which will render `index.html`. Any other URL will load the respective file on the computer.

```bash
Usage of alaybey:
  -f string
        folder to watch (default: current) (default ".")
  -i string
        index page to render on / (default "index.html")
  -p int
        port to serve (default 8003)
```

## Credits

https://github.com/schollz/browsersync
https://gist.github.com/fdrechsler/a20e8d2b8ff656db3bff9533e957be0c

## License

MIT