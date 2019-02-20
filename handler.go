package dvmweb

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"text/template"

	humanize "github.com/dustin/go-humanize"
	"github.com/gorilla/mux"
)

// fmap default functions for templates.
var fmap = template.FuncMap{
	"upper": strings.ToUpper,
	"ago":   humanize.Time,
	"clip": func(s string) string {
		if len(s) > 50 {
			return fmt.Sprintf("%s ...", s[:50])
		}
		return s
	},
}

// Handler implements HTTP request for reading, writing and rendering stories.
type Handler struct {
	App *App
}

// ReadHandler reads a story, given a random (image) identifier, e.g. "121403" or similar.
func (h *Handler) ReadHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rid := vars["rid"]
	t, err := template.New("read.html").Funcs(fmap).ParseFiles("templates/read.html")
	if t == nil || err != nil {
		log.Printf("failed or missing template: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var stories []Story
	err = h.App.db.Select(&stories, `
	SELECT id, imageid, text, language, created
	FROM story WHERE imageid = ?
	ORDER BY created DESC`, rid)
	if err != nil {
		log.Printf("SQL failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var data = struct {
		RandomIdentifier string
		Stories          []Story
	}{
		RandomIdentifier: rid,
		Stories:          stories,
	}
	if err := t.Execute(w, data); err != nil {
		log.Printf("render failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// WriteHandler creates a new story.
func (h *Handler) WriteHandler(w http.ResponseWriter, r *http.Request) {}

// StoryHandler links to a single story. One image can have multiple.
func (h *Handler) StoryHandler(w http.ResponseWriter, r *http.Request) {}

// AboutHandler render information about the app.
func (h *Handler) AboutHandler(w http.ResponseWriter, r *http.Request) {}

// IndexHandler render the home page.
func (h *Handler) IndexHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.New("index.html").Funcs(fmap).ParseFiles("templates/index.html")
	if t == nil || err != nil {
		log.Printf("failed or missing template: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var stories []Story
	err = h.App.db.Select(&stories, `
	SELECT id, imageid, text, language, created
	FROM story ORDER BY created DESC LIMIT 100`)
	if err != nil {
		log.Printf("SQL failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var data = struct {
		Stories               []Story
		RandomVideoIdentifier string
	}{
		Stories:               stories,
		RandomVideoIdentifier: "040223",
	}
	if err := t.Execute(w, data); err != nil {
		log.Printf("render failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
