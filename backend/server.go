package server

import (
	"bootstrap/backend/internal/httputil"
	"bootstrap/config"
	"compress/gzip"
	"database/sql"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"golang.org/x/net/html"
)

type Server struct {
	config *config.Config

	db *sql.DB

	// for /api routes
	router *mux.Router

	// for all other routes
	staticRouter *mux.Router

	// react serve
	reactPath  string
	reactIndex string

	// httpLogger        *log.Logger
	// httpLoggerFile    *os.File
	// http500Logger     *log.Logger
	// http500LoggerFile *os.File
}

func New(db *sql.DB, conf *config.Config) (*Server, error) {
	r := mux.NewRouter()

	s := &Server{
		router:       r,
		staticRouter: mux.NewRouter(),
		db:           db,
		config:       conf,
		reactPath:    "./frontend/dist/",
		reactIndex:   "index.html",
	}

	s.staticRouter.PathPrefix("/").HandlerFunc(s.serveSPA)

	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// beginT := time.Now()

	if r.URL.Path == "/robots.txt" {
		http.ServeFile(w, r, "./robots.txt")
	} else if r.URL.Path == "/manifest.json" {
		w.Header().Add("Cache-Control", "no-cache")
		http.ServeFile(w, r, "./frontend/dist/manifest.json")
	} else {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Add("Content-Type", "application/json; charset=UTF-8")
			w.Header().Add("Cache-Control", "no-store")
			httputil.GzipHandler(s.router).ServeHTTP(w, r)
		} else {
			s.staticRouter.ServeHTTP(w, r)
		}
	}

	// sid := "" // session id
	// if c, err := r.Cookie(s.config.SessionCookieName); err == nil {
	// 	sid = c.Value
	// }

	//took := time.Since(beginT)

	// logFields := []logField{
	// 	{name: "took", val: took},
	// 	{name: "url", val: r.URL},
	// 	{name: "ip", val: httputil.GetIP(r)},
	// 	{name: "method", val: r.Method},
	// 	{name: "sid", val: sid},
	// 	{name: "user-agent", val: r.Header.Get("User-Agent")},
	// }
	// s.httpLogger.Println(constructLogLine(logFields, ""))
	// if s.config.IsDevelopment && os.Getenv("NO_HTTP_LOG_LINE") != "true" {
	// 	logFields[len(logFields)-1].off = true
	// 	var color string
	// 	if took > time.Millisecond*10 {
	// 		color = "\033[0;33m"
	// 	}
	// 	log.Println(constructLogLine(logFields, color))
	// }
}

func (s *Server) serveSPA(w http.ResponseWriter, r *http.Request) {
	// Move incoming requests with a trailing slash to a url without it.
	if r.URL.Path != "/" && strings.HasSuffix(r.URL.Path, "/") {
		var u url.URL = *r.URL
		u.Path = strings.TrimSuffix(u.Path, "/")
		http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
		return
	}

	path, err := filepath.Abs(r.URL.Path)
	if err != nil {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	serveIndexFile := func() {
		file, err := os.Open(filepath.Join(s.reactPath, s.reactIndex))
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		doc, err := html.Parse(file)
		if err != nil {
			log.Fatal(err)
		}
		w.Header().Add("Cache-Control", "no-store")

		var writer io.Writer = w
		if httputil.AcceptEncoding(r.Header, "gzip") {
			gz := gzip.NewWriter(w)
			defer gz.Close()
			writer = gz
			w.Header().Add("Content-Encoding", "gzip")
			w.Header().Add("Content-Type", "text/html; charset=UTF-8")
		}
		html.Render(writer, doc)
	}

	if path == "/" {
		serveIndexFile()
		return
	}

	fpath := filepath.Join(s.reactPath, path)
	_, err = os.Stat(fpath)
	if os.IsNotExist(err) {
		serveIndexFile()
		return
	} else if err != nil {
		http.Error(w, "500: Internal server error", http.StatusInternalServerError)
		return
	}

	httputil.FileServer(http.Dir(s.reactPath)).ServeHTTP(w, r)
}
