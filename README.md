# AutoPrint

Ever wanted to check an URL periodically, and print out a copy if it's changed?


## Installation

```console
$ go install github.com/adammck/autoprint@latest
```

## Usage

```console
$ autoprint -h
usage: ./autoprint [options] <url>

options:
  -f    ignore etag to force download
  -n    don't print anything
  -v    enable verbose logging
```

For example:

```console
$ autoprint https://pdfobject.com/pdf/sample.pdf
Printing...

$ !!
Not modified.
```

Use cron or launchd or systemd or whatever to run it periodically.

## License

MIT.
