FILES = main.go crontab.go template.go bindata.go

test: bindata.go
	go test -v -cover

smoke: $(FILES)
	go run $(FILES) --help

build: $(FILES)
	gox -osarch="linux/amd64 darwin/amd64"

bindata.go:
	go-bindata -ignore=\.DS_Store templates/...

clean:
	rm cron2html* bindata.go
