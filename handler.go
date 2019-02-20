package dvmweb

import (
	"log"
	"net/http"
	"text/template"

	"github.com/gorilla/mux"
)

// Handler implements HTTP request for reading, writing and rendering stories.
type Handler struct {
	App *App
}

// Read reads a story, given a random (image) identifier, e.g. "121403" or similar.
func (h *Handler) Read(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rid := vars["rid"]
	t, err := template.New("read.html").ParseFiles("templates/read.html")
	if t == nil || err != nil {
		log.Printf("failed template: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var stories []Story
	err := h.app.db.Select(&stories, `
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
		RandomIdentifier: randomIdentifier,
		Stories:          stories,
	}
}

// Write creates a new story.
func (h *Handler) Write(w http.ResponseWriter, r *http.Request) {}

// Story links to a single story. One image can have multiple.
func (h *Handler) Story(w http.ResponseWriter, r *http.Request) {}

// About render information about the app.
func (h *Handler) About(w http.ResponseWriter, r *http.Request) {}

// Home render the home page.
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {}
