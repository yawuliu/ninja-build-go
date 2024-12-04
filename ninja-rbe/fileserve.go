package ninja_rbe

import (
	"expvar"
	"fmt"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/expvarhandler"
	"log"
	"net/http"
	"time"
)

// Various counters - see https://pkg.go.dev/expvar for details.
var (
	// Counter for total number of fs calls
	fsCalls = expvar.NewInt("fsCalls")

	// Counters for various response status codes
	fsOKResponses          = expvar.NewInt("fsOKResponses")
	fsNotModifiedResponses = expvar.NewInt("fsNotModifiedResponses")
	fsNotFoundResponses    = expvar.NewInt("fsNotFoundResponses")
	fsOtherResponses       = expvar.NewInt("fsOtherResponses")

	// Total size in bytes for OK response bodies served.
	fsResponseBodyBytes = expvar.NewInt("fsResponseBodyBytes")
)

func HandleUpload(ctx *fasthttp.RequestCtx) {
	output := string(ctx.FormValue("output"))
	commandHash := string(ctx.FormValue("command_hash"))
	startTime := string(ctx.FormValue("start_time"))
	endTime := string(ctx.FormValue("end_time"))
	mtime := string(ctx.FormValue("mtime"))
	header, err := ctx.FormFile("file")
	if err != nil {
		ctx.Error(err.Error(), http.StatusInternalServerError)
		return
	}
	storeName := fmt.Sprintf("%s_%s", commandHash, mtime)
	if err := fasthttp.SaveMultipartFile(header, storeName); err != nil {
		ctx.Error(err.Error(), http.StatusInternalServerError)
		return
	}
}

func updateFSCounters(ctx *fasthttp.RequestCtx) {
	// Increment the number of fsHandler calls.
	fsCalls.Add(1)

	// Update other stats counters
	resp := &ctx.Response
	switch resp.StatusCode() {
	case fasthttp.StatusOK:
		fsOKResponses.Add(1)
		fsResponseBodyBytes.Add(int64(resp.Header.ContentLength()))
	case fasthttp.StatusNotModified:
		fsNotModifiedResponses.Add(1)
	case fasthttp.StatusNotFound:
		fsNotFoundResponses.Add(1)
	default:
		fsOtherResponses.Add(1)
	}
}

func ServeFiles(addr, rootDir string, compress, byteRange, generateIndexPages, vhost bool) {
	// Setup FS handler
	fs := &fasthttp.FS{
		Root:               rootDir,
		IndexNames:         []string{"index.html"},
		GenerateIndexPages: generateIndexPages,
		Compress:           compress,
		AcceptByteRange:    byteRange,
	}
	if vhost {
		fs.PathRewrite = fasthttp.NewVHostPathRewriter(0)
	}
	fsHandler := fs.NewRequestHandler()
	// Create RequestHandler serving server stats on /stats and files
	// on other requested paths.
	// /stats output may be filtered using regexps. For example:
	//
	//   * /stats?r=fs will show only stats (expvars) containing 'fs'
	//     in their names.
	requestHandler := func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/stats":
			expvarhandler.ExpvarHandler(ctx)
		case "/upload":
			HandleUpload(ctx)
		default:
			fsHandler(ctx)
			updateFSCounters(ctx)
		}
	}
	// Start HTTP server.
	if len(addr) > 0 {
		log.Printf("Starting HTTP server on %q", addr)
		server := &fasthttp.Server{
			Handler:      requestHandler,
			ReadTimeout:  15 * time.Minute,
			WriteTimeout: 15 * time.Minute,
			Concurrency:  256 * 1024,
		}
		if err := server.ListenAndServe(addr); err != nil {
			log.Fatalf("error in ListenAndServe: %v", err)
		}
	}
	// Wait forever.
}
