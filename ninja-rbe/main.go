package ninja_rbe

import (
	"flag"
)

var (
	addr               = flag.String("addr", "localhost:8080", "TCP address to listen to")
	byteRange          = flag.Bool("byteRange", false, "Enables byte range requests if set to true")
	compress           = flag.Bool("compress", false, "Enables transparent response compression if set to true")
	dir                = flag.String("dir", "/usr/share/nginx/html", "Directory to serve static files from")
	generateIndexPages = flag.Bool("generateIndexPages", true, "Whether to generate directory index pages")
	vhost              = flag.Bool("vhost", false, "Enables virtual hosting by prepending the requested path with the requested hostname")
)

func main() {
	// Parse command-line flags.
	flag.Parse()
	go ServeFiles(*addr, *dir, *compress, *byteRange, *generateIndexPages, *vhost)
}
