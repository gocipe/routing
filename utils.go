package routing

import (
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

func lookupContent(root http.FileSystem, upath string) (http.File, os.FileInfo, bool) {
	var (
		err  error
		file http.File
		info os.FileInfo
	)

	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
	}

	upath = path.Clean(upath)

	if file, err = root.Open(upath); err != nil {
		return nil, nil, false
	}

	if info, err = file.Stat(); err != nil {
		return nil, nil, false
	}

	if info.IsDir() {
		upath += "/index.html"
		return lookupContent(root, upath+"/index.html")
	}

	return file, info, true
}

type fileHandlerWithFallback struct {
	root     http.FileSystem
	fallback http.File
}

// FileServerWithFallback returns an HTTP static fileserver with a default file fallback if requested url was not found
func FileServerWithFallback(root http.FileSystem, fallback http.File) http.Handler {
	return &fileHandlerWithFallback{root: root, fallback: fallback}
}

func (f *fileHandlerWithFallback) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		file http.File
		info os.FileInfo
		ok   bool
		err  error
	)

	file, info, ok = lookupContent(f.root, r.URL.Path)

	if !ok {
		file = f.fallback
		if info, err = file.Stat(); err == nil {
			ok = true
		}
	}

	if ok {
		http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("File not found and default could not be served."))
	}
}

type fileHandlerWithFallbackContent struct {
	root        http.FileSystem
	fallback    io.ReadSeeker
	filename    string
	contentType string
	modtime     time.Time
}

// FileServerWithFallbackContent returns an HTTP static fileserver with a default content fallback if requested url was not found
func FileServerWithFallbackContent(root http.FileSystem, fallback io.ReadSeeker, filename, contentType string, modtime time.Time) http.Handler {
	return &fileHandlerWithFallbackContent{root: root, fallback: fallback, filename: filename, contentType: contentType, modtime: modtime}
}

func (f *fileHandlerWithFallbackContent) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", f.contentType)
	http.ServeContent(w, r, f.filename, f.modtime, f.fallback)
}

type fileHandlerWithNotFoundHandler struct {
	root    http.FileSystem
	handler http.Handler
}

// FileServerWithNotFoundHandler returns an HTTP static fileserver with a custom http.Handler if requested url was not found
func FileServerWithNotFoundHandler(root http.FileSystem, handler http.Handler) http.Handler {
	return &fileHandlerWithNotFoundHandler{root: root, handler: handler}
}

func (f *fileHandlerWithNotFoundHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//TODO:
	//Lastoplace
	if r.URL.Path == "index.html" {
		f.handler.ServeHTTP(w, r)
	} else if file, info, ok := lookupContent(f.root, r.URL.Path); ok {
		http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	} else {
		f.handler.ServeHTTP(w, r)
	}
}
