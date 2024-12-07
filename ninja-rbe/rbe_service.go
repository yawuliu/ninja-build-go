package main

import (
	"context"
	"encoding/json"
	"expvar"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/expvarhandler"
	"log"
	"path/filepath"
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

	fsRootDir string
	fsServer  *fasthttp.Server
)

func HandleUpload(ctx *fasthttp.RequestCtx) {
	ctx.Response.Reset()
	output := string(ctx.FormValue("output"))
	commandHash := string(ctx.FormValue("command_hash"))
	startTime := string(ctx.FormValue("start_time"))
	endTime := string(ctx.FormValue("end_time"))
	mtime := string(ctx.FormValue("mtime"))
	outputHash := string(ctx.FormValue("output_hash"))
	instance := string(ctx.FormValue("instance"))
	expired_duration := string(ctx.FormValue("expired_duration"))
	header, err := ctx.FormFile("file")
	if err != nil {
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}
	exist, err := CheckCommandHashAndMtimeExist(commandHash, mtime)
	if err != nil {
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}
	if exist {
		ctx.Success("plain/text", []byte("already exists."))
		return
	}
	if err := fasthttp.SaveMultipartFile(header, filepath.Join(fsRootDir, outputHash)); err != nil {
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}
	err = InsertLogEntry(output, commandHash, startTime, endTime, mtime, outputHash,
		instance, expired_duration)
	if err != nil {
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}
	ctx.Success("plain/text", []byte("success"))
}

func HandleQuery(ctx *fasthttp.RequestCtx) {
	ctx.Response.Reset()
	instance := string(ctx.QueryArgs().Peek("instance"))
	output := string(ctx.QueryArgs().Peek("output"))
	commandHash := string(ctx.QueryArgs().Peek("command_hash"))
	mtime := string(ctx.QueryArgs().Peek("mtime"))
	found, err := FindCommandHashAndMtime(output, instance, commandHash, mtime)
	if err != nil {
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}
	buf, err := json.Marshal(found)
	if err != nil {
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}
	ctx.Success("application/json", buf)
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
	fsRootDir = rootDir
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
		case "/query":
			HandleQuery(ctx)
		default:
			fsHandler(ctx)
			updateFSCounters(ctx)
		}
	}
	// Start HTTP server.
	if len(addr) > 0 {
		log.Printf("Starting HTTP server on %q", addr)
		fsServer = &fasthttp.Server{
			Handler:      requestHandler,
			ReadTimeout:  15 * time.Minute,
			WriteTimeout: 15 * time.Minute,
			Concurrency:  256 * 1024,
		}
		if err := fsServer.ListenAndServe(addr); err != nil {
			log.Fatalf("error in ListenAndServe: %v", err)
		}
	}
	// Wait forever.
}

func shutdown(ctx context.Context) {
	CloseDb()
	StopScheduler()
	err := fsServer.ShutdownWithContext(ctx)
	if err != nil {
		log.Println(err)
	}
}
