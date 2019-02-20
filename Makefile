SHELL = /bin/bash

dvmweb: cmd/dvmweb/main.go
	go build -o $@ $<

clean:
	rm -f dvmweb

data.db:
		sqlite3 $@ < createdb.sql

