package dvmweb

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/disintegration/imaging"
	humanize "github.com/dustin/go-humanize"
	"github.com/gorilla/mux"
)

// fmap default functions for templates. TODO(miku): cleanup.
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
// Has access to application and static directory for assets, e.g. cached images.
type Handler struct {
	App       *App
	StaticDir string
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

// writeHeaderLog logs an error and writes HTTP status code to header.
func writeHeaderLog(w http.ResponseWriter, statusCode int, v interface{}) {
	log.Println(v)
	w.WriteHeader(statusCode)
}

// writeHeaderLog logs an error and writes HTTP status code to header.
func writeHeaderLogf(w http.ResponseWriter, statusCode int, s string, v ...interface{}) {
	log.Printf(s, v...)
	w.WriteHeader(statusCode)
}

// CacheImageRedirect is a helper handler, which creates an image on the fly, of just
// redirects to the static location of the image.
func (h *Handler) CacheImageRedirect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	iid := vars["iid"]
	if len(iid) != 6 {
		writeHeaderLogf(w, http.StatusBadRequest, "six digit image id expected, got %v", iid)
		return
	}
	// The cached file name.
	filename := filepath.Join(h.StaticDir, "cache", fmt.Sprintf("%s.jpg", iid))
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// Create cached version and put it under cache.

		var cimgs []*CategorizedImage
		var requested = []struct {
			category string
			imageid  string
		}{
			{"artifacts", iid[:2]},
			{"people", iid[2:4]},
			{"landscapes", iid[4:6]},
		}
		for _, req := range requested {
			img, err := h.App.Inventory.ByCategoryAndIdentifier(req.category, req.imageid)
			if err != nil {
				writeHeaderLogf(w, http.StatusNotFound, "cannot locate %v: %v", req.category, err)
				return
			}
			cimgs = append(cimgs, img)
		}

		// Resize to this height.
		resizeHeight := 300
		// Destination image.
		dst := imaging.New(960, 300, color.NRGBA{0, 0, 0, 0})

		// Iterate over images, resize and paste them into destination.
		for i, cimg := range cimgs {
			img, err := imaging.Open(cimg.Path)
			if err != nil {
				writeHeaderLogf(w, http.StatusInternalServerError, "cannot open image at: %v", cimg.Path)
				return
			}
			img = imaging.Resize(img, 0, resizeHeight, imaging.Lanczos)
			dst = imaging.Paste(dst, img, image.Pt(320*i, 0))
		}

		// Save to static cache dir, create the directory on the fly.
		cacheDir := filepath.Join(h.StaticDir, "cache")
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			writeHeaderLogf(w, http.StatusInternalServerError, "cannot create cache dir at %s", cacheDir)
			return
		}
		if err := imaging.Save(dst, filename); err != nil {
			writeHeaderLogf(w, http.StatusInternalServerError, "cannot save image to %s", filename)
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/static/cache/%s.jpg", iid), http.StatusSeeOther)
		return
	}
	// Redirect to static page.
	http.Redirect(w, r, fmt.Sprintf("/static/cache/%s.jpg", iid), http.StatusSeeOther)
}

// IndexHandler render the home page.
func (h *Handler) IndexHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.New("index.html").Funcs(fmap).ParseFiles("templates/index.html")
	if t == nil || err != nil {
		writeHeaderLogf(w, http.StatusInternalServerError, "failed or missing template: %v", err)
		return
	}

	var stories []Story
	err = h.App.db.Select(&stories, `
	SELECT id, imageid, text, language, created
	FROM story ORDER BY created DESC LIMIT 100`)
	if err != nil {
		writeHeaderLogf(w, http.StatusInternalServerError, "SQL failed: %v", err)
		return
	}

	// Video identifier, random image identifier.
	var vid, rid string

	// For frontpage animation.
	if vid, err = h.App.Inventory.RandomVideoIdentifier(); err != nil {
		writeHeaderLog(w, http.StatusInternalServerError, err)
		return
	}
	// For fallback image.
	if rid, err = h.App.Inventory.RandomImageIdentifier(); err != nil {
		writeHeaderLog(w, http.StatusInternalServerError, err)
		return
	}

	var data = struct {
		Stories               []Story
		RandomVideoIdentifier string
		RandomIdentifier      string
		RandomImageWithStory  string
	}{
		Stories:               stories,
		RandomVideoIdentifier: vid,
		RandomIdentifier:      rid,
		RandomImageWithStory:  "000000",
	}
	if err := t.Execute(w, data); err != nil {
		log.Printf("render failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
