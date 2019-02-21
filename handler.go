package dvmweb

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/disintegration/imaging"
	"github.com/gorilla/mux"
)

// fmap default functions for templates. TODO(miku): cleanup.
var fmap = template.FuncMap{
	"upper": strings.ToUpper,
	"datefmt": func(t time.Time) string {
		return t.Format("02.01.2006 15:04")
	},
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
	mu  sync.Mutex // Lock app and database access.
	App *App

	StaticDir string
}

// ReadHandler reads a story, given a random (image) identifier, e.g. "121403" or similar.
func (h *Handler) ReadHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	iid := vars["iid"]
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
	ORDER BY created DESC`, iid)
	if err != nil {
		log.Printf("SQL failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var data = struct {
		RandomIdentifier string
		Stories          []Story
	}{
		RandomIdentifier: iid,
		Stories:          stories,
	}
	if err := t.Execute(w, data); err != nil {
		log.Printf("render failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// WriteHandler creates a new story.
func (h *Handler) WriteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	iid := vars["iid"]

	if r.Method == "POST" {
		// Save new story to database.
		h.mu.Lock()
		defer h.mu.Unlock()

		// Parse url parameters passed, then parse the response packet for the
		// POST body (request body) attention: If you do not call ParseForm
		// method, the following data can not be obtained form.
		r.ParseForm()

		text := strings.TrimSpace(r.Form.Get("story"))
		language := r.Form.Get("language") // Assume form has sane default.

		if len(text) == 0 {
			writeHeaderLog(w, http.StatusNoContent, "no content")
			return
		}
		if len(text) > 10000 {
			writeHeaderLog(w, http.StatusBadRequest, "body exceeds limit")
			return
		}
		// TODO(miku): Add spam detector from https://git.io/fhFUf.

		// The ultimative rate limiter. Limits the amount postable to about
		// 400M per day. TODO(miku): Lookup IP address and send back a "you are
		// doing this too much" or similar.
		time.Sleep(2 * time.Second)

		stmt := `INSERT INTO story (imageid, text, language, ip, flagged) values (?, ?, ?, ?, ?)`
		result, err := h.App.db.Exec(stmt, iid, text, language, r.RemoteAddr, false)
		if err != nil {
			writeHeaderLogf(w, http.StatusInternalServerError, "insert failed: %v", err)
			return
		}
		lid, err := result.LastInsertId()
		if err != nil {
			writeHeaderLogf(w, http.StatusInternalServerError, "failed to get last insert id: %v", err)
			return
		}
		log.Printf("last insert id was: %v", lid)
		http.Redirect(w, r, fmt.Sprintf("/r/%s", iid), http.StatusSeeOther)
		return
	}

	// Render form.
	t, err := template.New("write.html").Funcs(fmap).ParseFiles("templates/write.html")
	if t == nil || err != nil {
		log.Printf("failed or missing template: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var data = struct {
		RandomIdentifier string
	}{
		RandomIdentifier: iid,
	}
	if err := t.Execute(w, data); err != nil {
		log.Printf("render failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// StoryHandler links to a single story. One image can have multiple.
func (h *Handler) StoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	identifier, err := strconv.Atoi(vars["id"])
	if err != nil {
		writeHeaderLog(w, http.StatusBadRequest, err)
		return
	}
	t, err := template.New("story.html").Funcs(fmap).ParseFiles("templates/story.html")
	if t == nil || err != nil {
		log.Printf("failed or missing template: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var story Story
	err = h.App.db.Get(&story, `
	SELECT id, imageid, text, language, created
	FROM story WHERE id = ? LIMIT 1`, identifier)
	if err != nil {
		writeHeaderLogf(w, http.StatusInternalServerError, "SQL failed: %v", err)
		return
	}
	// No story.
	if story.Text == "" {
		writeHeaderLogf(w, http.StatusNotFound, "missing story: %d", identifier)
		return
	}
	var data = struct {
		RandomIdentifier string
		Story            Story
	}{
		RandomIdentifier: story.ImageIdentifier,
		Story:            story,
	}
	if err := t.Execute(w, data); err != nil {
		log.Printf("template err: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// AboutHandler render information about the app.
func (h *Handler) AboutHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.New("about.html").Funcs(fmap).ParseFiles("templates/about.html")
	if t == nil || err != nil {
		log.Printf("failed or missing template: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Video identifier, random image identifier.
	var vid, rid string

	if vid, err = h.App.Inventory.RandomVideoIdentifier(); err != nil {
		writeHeaderLog(w, http.StatusInternalServerError, err)
		return
	}
	if rid, err = h.App.Inventory.RandomImageIdentifier(); err != nil {
		writeHeaderLog(w, http.StatusInternalServerError, err)
		return
	}

	var data = struct {
		RandomVideoIdentifier string
		RandomIdentifier      string
	}{
		RandomVideoIdentifier: vid,
		RandomIdentifier:      rid,
	}
	if err := t.Execute(w, data); err != nil {
		log.Printf("render failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
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

	// Fallback to some image.
	riws := "000000"
	if len(stories) == 0 {
		riws = stories[rand.Intn(len(stories))].ImageIdentifier
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
		RandomImageWithStory:  riws,
	}
	if err := t.Execute(w, data); err != nil {
		log.Printf("render failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

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
