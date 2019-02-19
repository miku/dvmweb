dvmweb: cmd/dvmweb/main.go
	go build -o $@ $<

clean:
	rm -f dvmweb

