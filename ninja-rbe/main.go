package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"
)

var (
	dbName             = flag.String("dbName", "ninja.db", "ninja remote db name.")
	addr               = flag.String("addr", "localhost:8080", "TCP address to listen to")
	byteRange          = flag.Bool("byteRange", false, "Enables byte range requests if set to true")
	compress           = flag.Bool("compress", false, "Enables transparent response compression if set to true")
	dir                = flag.String("dir", "html", "Directory to serve static files from")
	generateIndexPages = flag.Bool("generateIndexPages", true, "Whether to generate directory index pages")
	vhost              = flag.Bool("vhost", false, "Enables virtual hosting by prepending the requested path with the requested hostname")
)

func main() {
	// Parse command-line flags.
	flag.Parse()
	dbPath := filepath.Join(filepath.Dir(os.Args[0]), *dbName)
	err := OpenDb(dbPath)
	if err != nil {
		panic(err)
	}
	go ServeFiles(*addr, *dir, *compress, *byteRange, *generateIndexPages, *vhost)
	// Make a signal channel. Register SIGINT.
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)

	// Wait for the signal.
	<-sigch

	fmt.Println("Interrupted. Exiting.")
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	shutdown(ctx)
}
