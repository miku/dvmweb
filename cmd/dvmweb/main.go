package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/miku/dvmweb"
)

var (
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
	fmt.Println(app)
}
