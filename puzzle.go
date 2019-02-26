package dvmweb

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// CategorizedImage belongs to a category, path records the absolute path. The
// identifier is just the basename of the image, e.g. 12 for 12.jpg file.
type CategorizedImage struct {
	Identifier string
	Path       string
	Category   string
}

// Inventory make images and videos accessible in various ways. The slices
// contain the absolute path and a bit of derived information. This struct
// exposes a few functions, that help to locate images by category, by
// identifier and the like. TODO(miku): maybe categorize videos as well.
type Inventory struct {
	Images []CategorizedImage
	Videos []string
}

// videoIdentifier returns the identifier of a video, given its path. Go from
// /some/path/dvm-010303.mp4 to 010303.
func videoIdentifier(p string) string {
	b := strings.Replace(path.Base(p), "dvm-", "", -1)
	return strings.Replace(b, path.Ext(b), "", -1)
}

// NumAssets returns the total number of items in inventory.
func (inv *Inventory) NumAssets() int {
	return len(inv.Images) + len(inv.Videos)
}

// ByCategoryAndIdentifier returns the image descriptor for a given category
// and identifier.
func (inv *Inventory) ByCategoryAndIdentifier(category, identifier string) (*CategorizedImage, error) {
	for _, img := range inv.Images {
		if img.Category == category && img.Identifier == identifier {
			return &img, nil
		}
	}
	return nil, fmt.Errorf("image not found: %s/%s", category, identifier)
}

// RandomImageIdentifier returns a random composite image identifier from
// fixed categories (artifacts, people, landscape).
func (inv *Inventory) RandomImageIdentifier() (rid string, err error) {
	artifacts := inv.ByCategory("artifacts")
	people := inv.ByCategory("people")
	landscapes := inv.ByCategory("landscapes")

	if len(artifacts) == 0 || len(people) == 0 || len(landscapes) == 0 {
		return "", fmt.Errorf("incomplete image dirs, missing at one category")
	}

	return fmt.Sprintf("%s%s%s",
		artifacts[rand.Intn(len(artifacts))].Identifier,
		artifacts[rand.Intn(len(people))].Identifier,
		artifacts[rand.Intn(len(landscapes))].Identifier), nil
}

// ByCategory returns image paths by category, empty list, if the category does not exist.
func (inv *Inventory) ByCategory(category string) (images []CategorizedImage) {
	for _, img := range inv.Images {
		if img.Category == category {
			images = append(images, img)
		}
	}
	return
}

// RandomVideoIdentifier returns the id to a random video.
func (inv *Inventory) RandomVideoIdentifier() (vid string, err error) {
	if len(inv.Videos) == 0 {
		return "", fmt.Errorf("no videos found")
	}
	return videoIdentifier(inv.Videos[rand.Intn(len(inv.Videos))]), nil
}

// Categories returns the unique image categories.
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

// Ok checks if the inventory is somewhat usable. Also, we want three categories at the moment.
func (inv *Inventory) Ok() bool {
	return len(inv.Images) > 10 && len(inv.Videos) > 0 && len(inv.Categories()) == 3
}

// Story describes a minimal story.
type Story struct {
	Identifier      int       `db:"id"`
	ImageIdentifier string    `db:"imageid"`
	Text            string    `db:"text"`
	Language        string    `db:"language"`
	Created         time.Time `db:"created"`
}

// App configuration and data access layer.
type App struct {
	db        *sqlx.DB
	videosDir string
	imagesDir string
	Inventory *Inventory
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

// createInventory fixes images and video paths in an inventory.
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
		Inventory: inv,
	}, nil
}

func (app *App) String() string {
	return fmt.Sprintf("app with %d images in %d categories and %d videos",
		len(app.Inventory.Images),
		len(app.Inventory.Categories()),
		len(app.Inventory.Videos))
}
