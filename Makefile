FILES = main.go crontab.go template.go

test:
	go test -v -cover

smoke:
	go run $(FILES) --help

build:
	go build -o cron2html $(FILES)

clean:
	rm cron2html
