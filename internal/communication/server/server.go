package server

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	//"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/mux"
	core "github.com/miphilipp/devchat-server/internal"
	"github.com/miphilipp/devchat-server/internal/communication/session"
	"github.com/miphilipp/devchat-server/internal/communication/websocket"
	"github.com/miphilipp/devchat-server/internal/conversations"
	"github.com/miphilipp/devchat-server/internal/messaging"
	"github.com/miphilipp/devchat-server/internal/user"
	"github.com/throttled/throttled"
	"golang.org/x/text/language"
)

var matcher = language.NewMatcher([]language.Tag{
	language.German,
	language.English,
})

type ServerConfig struct {
	Addr                     string
	IndexFileName            string
	AssetsFolder             string
	AvatarFolder             string
	MediaFolder              string
	RootURL                  string
	AllowedRequestsPerMinute int
	MediaTokenSecret         []byte
	Webpages                 []string
}

type Webserver struct {
	config ServerConfig
	router *mux.Router
	Server *http.Server

	logger log.Logger

	socket  *websocket.Server
	session *session.Manager

	userService         user.Service
	conversationService conversations.Service
	messageService      messaging.Service
}

func (s *Webserver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path, err := filepath.Abs(r.URL.Path)
	if err != nil {
		level.Error(s.logger).Log("handler", "default", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// prepend the path with the path to the static directory
	path = filepath.Join(s.config.AssetsFolder, path)

	// check whether a file exists at the given path
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		// file does not exist, serve index.html
		if containsPath(s.config.Webpages, r.URL.Path) {
			if pusher, ok := w.(http.Pusher); ok {
				s.pushFiles(pusher, r)
			}
			w.Header().Set("Cache-Control", "no-cache")
			http.ServeFile(w, r, filepath.Join(s.config.AssetsFolder, s.config.IndexFileName))
		} else {
			writeJSONError(w, core.ErrRessourceDoesNotExist, http.StatusNotFound)
		}
		return
	} else if err != nil {
		// if we got an error (that wasn't that the file doesn't exist) stating the
		// file, return a 500 internal server error and stop
		level.Error(s.logger).Log("handler", "default", "err", err)
		writeJSONError(w, core.ErrUnknownError, http.StatusNotFound)
		return
	}

	// otherwise, use http.FileServer to serve the static dir
	w.Header().Set("Cache-Control", "max-age=31536000")
	http.FileServer(http.Dir(s.config.AssetsFolder)).ServeHTTP(w, r)
}

func (s *Webserver) pushFiles(pusher http.Pusher, r *http.Request) {
	options := &http.PushOptions{
		Header: http.Header{
			"Accept-Encoding": r.Header["Accept-Encoding"],
		},
	}
	cssPath := filepath.Join(s.config.AssetsFolder, "css")
	cssFiles, _ := ioutil.ReadDir(cssPath)
	for _, file := range cssFiles {
		path, _ := filepath.Abs(filepath.Join(cssPath, file.Name()))
		if err := pusher.Push(path, options); err != nil {
			s.logger.Log("PushFail", err)
		}
	}

	jsPath := filepath.Join(s.config.AssetsFolder, "js")
	jsFiles, _ := ioutil.ReadDir(jsPath)
	for _, file := range jsFiles {
		path, _ := filepath.Abs(filepath.Join(cssPath, file.Name()))
		if err := pusher.Push(path, options); err != nil {
			s.logger.Log("PushFail", err)
		}
	}
}

func limitSize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 32<<20+1024)
		next.ServeHTTP(w, r)
	})
}

func parseLanguage(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept := r.Header.Get("Accept-Language")
		tag, _ := language.MatchStrings(matcher, accept)
		copiedRequest := r.WithContext(context.WithValue(r.Context(), "Language", tag))
		next.ServeHTTP(w, copiedRequest)
	})
}

// New creates a new instance of type Webserver
func New(
	cfg ServerConfig,
	userService user.Service,
	cService conversations.Service,
	mService messaging.Service,
	socket *websocket.Server,
	session *session.Manager,
	limiterStore throttled.GCRAStore,
	logger log.Logger) *Webserver {

	router := mux.NewRouter()
	srv := &http.Server{
		Handler:      router,
		Addr:         cfg.Addr,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	app := &Webserver{
		config:              cfg,
		router:              router,
		Server:              srv,
		userService:         userService,
		conversationService: cService,
		messageService:      mService,
		logger:              logger,
		socket:              socket,
		session:             session,
	}

	if cfg.AllowedRequestsPerMinute == 0 {
		cfg.AllowedRequestsPerMinute = 10
	}

	quota := throttled.RateQuota{MaxRate: throttled.PerMin(cfg.AllowedRequestsPerMinute), MaxBurst: 5}
	rateLimiter, err := throttled.NewGCRARateLimiter(limiterStore, quota)
	if err != nil {
		return nil
	}

	httpRateLimiter := throttled.HTTPRateLimiter{
		RateLimiter: rateLimiter,
		VaryBy:      &throttled.VaryBy{Path: true, Method: true, RemoteAddr: true},
	}

	router.Use(limitSize)
	router.Use(parseLanguage)
	router.Use(httpRateLimiter.RateLimit)

	return app
}

// SetupFileServer sets up file server handlers for all webpages and its subpages.
func (s *Webserver) SetupFileServer() {
	for _, path := range s.config.Webpages {
		s.router.PathPrefix(path).Handler(s).Methods(http.MethodGet)
	}
}

func containsPath(paths []string, path string) bool {
	for _, p := range paths {
		if p == path {
			return true
		}
	}
	return false
}
