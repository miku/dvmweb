package dvmweb

import (
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// CategorizedImage belongs to a category, path records the absolute path. The
// identifier is just the basename of the image, e.g. 12 for 12.jpg file.
type CategorizedImage struct {
	Identifier string
	Path       string
	Category   string
}

// Inventory lists all assets.
type Inventory struct {
	Images []CategorizedImage
	Videos []string
}

// NumAssets returns the total number of items in inventory.
func (inv *Inventory) NumAssets() int {
	return len(inv.Images) + len(inv.Videos)
}

// ByCategory returns image paths by category, empty list, if the category does not exist.
func (inv *Inventory) ByCategory(category string) (paths []string) {
	for _, img := range inv.Images {
		if img.Category == category {
			paths = append(paths, img.Path)
		}
	}
	return
}

func (inv *Inventory) Categories() (categories []string) {
	m := make(map[string]struct{})
	for _, img := range inv.Images {
		m[img.Category] = struct{}{}
	}
	for k := range m {
		categories = append(categories, k)
	}
	return
}

// Ok checks if the inventory is somewhat usable.
func (inv *Inventory) Ok() bool {
	return len(inv.Images) > 10 && len(inv.Videos) > -1
}

// Story describes a minimal story.
type Story struct {
	Identifier      int       `db:"id"`
	ImageIdentifier string    `db:"imageid"`
	Text            string    `db:"text"`
	Language        string    `db:"language"`
	Created         time.Time `db:"created"`
}

// Application configuration and data access layer.
type App struct {
	db        *sqlx.DB
	videosDir string
	imagesDir string
	inventory *Inventory
}

// subdirNames returns the names of direct subfolders.
func subdirNames(dir string) (subdirs []string, err error) {
	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		return
	}
	for _, fi := range fis {
		if !fi.IsDir() {
			continue
		}
		subdirs = append(subdirs, fi.Name())
	}
	return
}

// findImages will return a list of absolute paths for images of a given
// category, which is just a subdirectory of the root dir.
func findImages(root, category string) (filenames []string, err error) {
	cdir := filepath.Join(root, category)
	fis, err := ioutil.ReadDir(cdir)
	if err != nil {
		return
	}
	for _, fi := range fis {
		if fi.IsDir() {
			continue
		}
		if !strings.HasSuffix(fi.Name(), ".jpg") {
			continue
		}
		filenames = append(filenames, filepath.Join(cdir, fi.Name()))
	}
	return
}

// createInventory fixes images and video paths.
func createInventory(imagesDir, videosDir string) (*Inventory, error) {
	subdirs, err := subdirNames(imagesDir)
	if err != nil {
		return nil, err
	}
	inv := Inventory{}

	// Read and categorize images.
	for _, c := range subdirs {
		files, err := findImages(imagesDir, c)
		if err != nil {
			return nil, err
		}
		for _, p := range files {
			inv.Images = append(inv.Images, CategorizedImage{
				Identifier: strings.Replace(path.Base(p), path.Ext(p), "", -1),
				Path:       p,
				Category:   c,
			})
		}
	}

	fis, err := ioutil.ReadDir(videosDir)
	if err != nil {
		return nil, err
	}
	for _, fi := range fis {
		// XXX: allow for various formats.
		if !strings.HasSuffix(fi.Name(), ".mp4") {
			continue
		}
		inv.Videos = append(inv.Videos, filepath.Join(videosDir, fi.Name()))
	}
	return &inv, nil
}

// New create a new web app given a data source and some static directories.
func New(dsn, imagesDir, videosDir string) (*App, error) {
	log.Println("creating inventory")
	inv, err := createInventory(imagesDir, videosDir)
	if err != nil {
		return nil, err
	}
	if !inv.Ok() {
		return nil, fmt.Errorf("incomplete inventory")
	}
	db, err := sqlx.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	return &App{
		db:        db,
		videosDir: videosDir,
		imagesDir: imagesDir,
		inventory: inv,
	}, nil
}

func (app *App) String() string {
	return fmt.Sprintf("app with %d images in %d categories and %d videos",
		len(app.inventory.Images),
		len(app.inventory.Categories()),
		len(app.inventory.Videos))
}
