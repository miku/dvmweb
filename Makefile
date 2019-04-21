SHELL = /bin/bash
PKGNAME = dvmweb
TARGETS = dvmweb

dvmweb: cmd/dvmweb/main.go
	go get ./...
	go build -ldflags "-X main.version=`git rev-parse --short HEAD`"  -o $@ $<

clean:
	rm -f dvmweb

data.db:
		sqlite3 $@ < createdb.sql

deb: dvmweb
	mkdir -p packaging/deb/$(PKGNAME)/usr/sbin
	cp $(TARGETS) packaging/deb/$(PKGNAME)/usr/sbin
	mkdir -p packaging/deb/$(PKGNAME)/usr/lib/systemd/system
	cp packaging/dvmweb.service packaging/deb/$(PKGNAME)/usr/lib/systemd/system/
	cd packaging/deb && fakeroot dpkg-deb --build $(PKGNAME) .
	mv packaging/deb/$(PKGNAME)_*.deb .

