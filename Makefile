FILES = main.go crontab.go template.go

test:
	go test -v -cover

smoke:
	go run $(FILES) --help

build:
	gox -osarch="linux/amd64 darwin/amd64"

clean:
	rm cron2html
