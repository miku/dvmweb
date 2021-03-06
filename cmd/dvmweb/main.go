package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/miku/dvmweb"

	_ "github.com/mattn/go-sqlite3"
)

var (
	listen       = flag.String("listen", "0.0.0.0:3000", "hostport to listen on")
	logfile      = flag.String("log", "", "logfile or stderr if empty")
	dsn          = flag.String("dsn", "data.db", "data source name, e.g. sqlite3 path")
	imagesDir    = flag.String("i", "static/images", "path to images, one subdirectory per category")
	videosDir    = flag.String("v", "static/videos", "path to videos")
	staticDir    = flag.String("s", "static", "static dir")
	templatesDir = flag.String("t", "templates", "template dir")

	version = "dev"
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

	// Make sure, static dir ends with a slash.
	*staticDir = fmt.Sprintf("%s/", strings.TrimRight(*staticDir, "/"))

	// Handler implement HTTP handlers for app.
	h := dvmweb.Handler{
		App:          app,
		StaticDir:    *staticDir,
		TemplatesDir: *templatesDir,
		Version:      version,
	}

	// Setup routes.
	r := mux.NewRouter()

	// Server static assets of defined dir.
	fs := http.FileServer(http.Dir(*staticDir))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static", fs))
	// Handlers.
	r.HandleFunc("/c/{iid}.jpg", h.CacheImageRedirect)
	r.HandleFunc("/w/{iid}", h.WriteHandler)
	r.HandleFunc("/r/{iid}", h.ReadHandler)
	r.HandleFunc("/s/{id}", h.StoryHandler)
	r.HandleFunc("/", h.IndexHandler)
	r.HandleFunc("/rand", h.RandomRead)
	r.HandleFunc("/about", h.AboutHandler)
	r.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/robots.txt", 302)
	})
	r.HandleFunc("/humans.txt", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/humans.txt", 302)
	})
	r.NotFoundHandler = http.HandlerFunc(h.NotFoundHandler)
	http.Handle("/", r)

	// Add middleware.
	logr := handlers.LoggingHandler(logw, r)

	log.Printf("starting server at http://%v", *listen)
	log.Fatal(http.ListenAndServe(*listen, logr))
}
