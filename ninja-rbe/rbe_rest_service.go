package main

import (
	"cmp"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/valyala/fasthttp"
	"github.com/zeebo/blake3"
	"log"
	"ninja-build-go/model"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// Various counters - see https://pkg.go.dev/expvar for details.
var (
	// Counter for total number of fs calls
	// fsCalls = expvar.NewInt("fsCalls")

	// Counters for various response status codes
	//fsOKResponses          = expvar.NewInt("fsOKResponses")
	//fsNotModifiedResponses = expvar.NewInt("fsNotModifiedResponses")
	//fsNotFoundResponses    = expvar.NewInt("fsNotFoundResponses")
	//fsOtherResponses       = expvar.NewInt("fsOtherResponses")

	// Total size in bytes for OK response bodies served.
	//fsResponseBodyBytes = expvar.NewInt("fsResponseBodyBytes")

	fsRootDir string
	fsServer  *fasthttp.Server
)

func ParseLogEntry(ctx *fasthttp.RequestCtx) (*model.RbeLogEntry, error) {
	body := ctx.FormValue("body")
	base64Buf := make([]byte, base64.StdEncoding.DecodedLen(len(body)))
	_, err := base64.StdEncoding.Decode(base64Buf, body)
	if err != nil {
		return nil, err
	}
	var entry model.RbeLogEntry
	err = json.Unmarshal(base64Buf, &entry)
	if err != nil {
		return nil, err
	}
	expired_duration_str := string(ctx.FormValue("expired_duration"))
	expired_duration := 5 * time.Minute
	if expired_duration_str != "" {
		expired_duration, _ = time.ParseDuration(expired_duration_str)
	}
	now := time.Now()
	created_at := now.Unix()
	last_access := created_at
	entry.CreatedAt = created_at
	entry.LastAccess = last_access
	entry.ExpiredDuration = int64(expired_duration)
	return &entry, nil
}

func HashEntry(entry *model.RbeLogEntry) string {
	h := blake3.New()
	h.WriteString(fmt.Sprintf("n:%s,%s,%s, %s\n", entry.Output,
		entry.CommandHash, entry.OutputHash, entry.InputHash))
	for _, dep := range entry.Deps {
		h.WriteString(fmt.Sprintf("d:%s,%s\n", dep.FilePath, dep.FileHash))
	}
	return string(h.Sum(nil))
}

func HandleUpload(ctx *fasthttp.RequestCtx) {
	ctx.Response.Reset()
	header, err := ctx.FormFile("file")
	if err != nil {
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}
	entry, err := ParseLogEntry(ctx)
	//==
	slices.SortFunc(entry.Deps, func(a, b *model.DepsEntry) int {
		return cmp.Compare(a.FilePath, b.FilePath)
	})
	paramsHash := HashEntry(entry)
	//==
	exist, err := CheckEntryExist(paramsHash)
	if err != nil {
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}
	entry.ParamsHash = paramsHash
	if exist {
		ctx.Success("plain/text", []byte("already exists."))
		return
	}
	if err := fasthttp.SaveMultipartFile(header, filepath.Join(fsRootDir, paramsHash)); err != nil {
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}
	//==
	err = SaveLogEntry(entry)
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
	input_hash := string(ctx.QueryArgs().Peek("input_hash"))
	potentialRecords, err := FindPotentialCacheRecords(instance, output, commandHash, input_hash)
	if err != nil {
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}
	buf, err := json.Marshal(potentialRecords)
	if err != nil {
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}
	ctx.Success("application/json", buf)
}

func UpdateRecordLastAccess(ctx *fasthttp.RequestCtx) {
	path := string(ctx.Path())
	paths := strings.Split(path, "/")
	if len(paths) != 2 {
		return
	}
	paramsHash := paths[1]
	err := UpdateFileAccess(paramsHash)
	if err != nil {
		fmt.Println(err)
	}
}

//func updateFSCounters(ctx *fasthttp.RequestCtx) {
//	// Increment the number of fsHandler calls.
//	fsCalls.Add(1)
//
//	// Update other stats counters
//	resp := &ctx.Response
//	switch resp.StatusCode() {
//	case fasthttp.StatusOK:
//		fsOKResponses.Add(1)
//		fsResponseBodyBytes.Add(int64(resp.Header.ContentLength()))
//	case fasthttp.StatusNotModified:
//		fsNotModifiedResponses.Add(1)
//	case fasthttp.StatusNotFound:
//		fsNotFoundResponses.Add(1)
//	default:
//		fsOtherResponses.Add(1)
//	}
//}

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
		//case "/stats":
		//	expvarhandler.ExpvarHandler(ctx)
		case "/upload":
			HandleUpload(ctx)
		case "/query":
			HandleQuery(ctx)
		default:
			fsHandler(ctx)
			UpdateRecordLastAccess(ctx)
			//updateFSCounters(ctx)
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
