# CRON to HTML
[![Build Status](https://travis-ci.org/Barzahlen/cron2html.svg)](https://travis-ci.org/Barzahlen/cron2html)

A CRON to HTML documentation generator for multiple servers written in golang.

## Installation

Just download the [cron2html](https://github.com/Barzahlen/cron2html/releases/download/v1.0.0/cron2html) binary.

Alternatively install it with

```
$ go get github.com/Barzahlen/cron2html
```

## Usage

Cron2html assumes that you authenticate to all servers via a publickey that is available in your local SSH agent. To add your default SSH publickey to SSH agent, enter `ssh-add`.

```
cron2html --help
cron2html -o cronjobs.html server1 server2
```

## Example output
![Example output](https://raw.github.com/Barzahlen/cron2html/master/example.png)

## Contributing
This is an open source project and your contribution is very much appreciated.

1. Check for open issues or open a fresh issue to start a discussion around a feature idea or a bug.
2. Fork the repository on Github and make your changes on the **develop** branch (or branch off of it).
3. Send a pull request (with the **develop** branch as the target).


## Changelog
See [CHANGELOG.md](changelog.md)


## License
cron2html is available under the GPL v3 license. See the [LICENSE](LICENSE) file for more info.
