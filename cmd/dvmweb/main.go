package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/miku/dvmweb"
)

var (
	listen    = flag.String("listen", "localhost:3000", "hostport to listen on")
	logfile   = flag.String("log", "", "logfile or stderr if empty")
	dsn       = flag.String("dsn", "data.db", "data source name, e.g. sqlite3 path")
	imagesDir = flag.String("i", "static/images", "path to images, one subdirectory per category")
	videosDir = flag.String("v", "static/videos", "path to videos")
)

func main() {
	flag.Parse()

	app, err := dvmweb.New(*dsn, *imagesDir, *videosDir)
	if err != nil {
		log.Fatal(err)
	}

	var logw = os.Stdout

	if *logfile != "" {
		f, err := os.OpenFile(*logfile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Fatal(err)
		}
		logw = f
		defer f.Close()
	}

	// Handler implement HTTP handlers for app.
	h := dvmweb.Handler{App: app}

	// Setup routes.
	r := mux.NewRouter()
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static", http.FileServer(http.Dir("static/"))))
	r.HandleFunc("/", h.IndexHandler)
	r.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/robots.txt", 302)
	})
	http.Handle("/", r)

	// Add middleware.
	logr := handlers.LoggingHandler(logw, r)

	log.Printf("starting server at http://%v", *listen)
	log.Fatal(http.ListenAndServe(*listen, logr))
}
