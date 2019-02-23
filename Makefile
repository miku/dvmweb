SHELL = /bin/bash

dvmweb: cmd/dvmweb/main.go
	go get ./...
	go build -ldflags "-X main.version=`git rev-parse --short HEAD`"  -o $@ $<

clean:
	rm -f dvmweb

data.db:
		sqlite3 $@ < createdb.sql

