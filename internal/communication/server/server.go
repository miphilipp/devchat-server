package server

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
	AllowedRequestsPerMinute int
	MediaTokenSecret         []byte
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

	actualWebpages []string
}

func (s Webserver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path, err := filepath.Abs(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// prepend the path with the path to the static directory
	path = filepath.Join(s.config.AssetsFolder, path)

	// check whether a file exists at the given path
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		// file does not exist, serve index.html
		if containsPath(s.actualWebpages, r.URL.Path) {
			http.ServeFile(w, r, filepath.Join(s.config.AssetsFolder, s.config.IndexFileName))
		} else {
			writeJSONError(w, core.ErrRessourceDoesNotExist, http.StatusNotFound)
		}
		return
	} else if err != nil {
		// if we got an error (that wasn't that the file doesn't exist) stating the
		// file, return a 500 internal server error and stop
		level.Error(s.logger).Log("handler", "default", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// otherwise, use http.FileServer to serve the static dir
	http.FileServer(http.Dir(s.config.AssetsFolder)).ServeHTTP(w, r)
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

// New erstellt eine neue Instanz vom Typ Webserver
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
		actualWebpages:      []string{"/login", "/forgot", "/", "/preferences", "/confirm"},
	}

	if cfg.AllowedRequestsPerMinute == 0 {
		cfg.AllowedRequestsPerMinute = 10
	}

	quota := throttled.RateQuota{MaxRate: throttled.PerMin(cfg.AllowedRequestsPerMinute), MaxBurst: 3}
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

func (s *Webserver) getMediaToken(writer http.ResponseWriter, request *http.Request) {
	ttl := time.Hour * 1
	token, err := session.GetMediaToken(ttl, s.config.MediaTokenSecret)
	if err != nil {
		writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		return
	}

	reply := struct {
		Token      string `json:"token"`
		Expiration int64  `json:"expiration"`
	}{
		Token:      token,
		Expiration: time.Now().Add(ttl).Unix(),
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(reply)
}

func (s *Webserver) generateMediaAuthenticationMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			token := request.FormValue("token")
			if token == "" {
				fieldError := core.NewInvalidValueError("token")
				writeJSONError(writer, fieldError, http.StatusBadRequest)
				return
			}

			ok, _ := session.VerifyMediaToken(token, s.config.MediaTokenSecret)
			if ok {
				next.ServeHTTP(writer, request)
			} else {
				checkForAPIError(core.ErrAuthFailed, writer)
			}
		})
	}
}

// SetupFileServer sets up file server handlers for all webpages and its subpages.
func (s Webserver) SetupFileServer() {
	for _, path := range s.actualWebpages {
		s.router.PathPrefix(path).Handler(s).Methods(http.MethodGet)
	}
}

func (s *Webserver) logout(writer http.ResponseWriter, request *http.Request) {
	tokenString := request.Header.Get("Authorization")
	if len(tokenString) == 0 {
		http.Error(writer, "Missing header", http.StatusBadRequest)
		return
	}

	tokenString = strings.Replace(tokenString, "Bearer ", "", 1)
	err := s.session.InvlidateToken(tokenString)
	if err != nil {
		level.Error(s.logger).Log("handler", "logout", "err", err)
		http.Error(writer, "Forbidden", http.StatusInternalServerError)
		return
	}
	writer.WriteHeader(http.StatusOK)
}

func containsPath(paths []string, path string) bool {
	for _, p := range paths {
		if p == path {
			return true
		}
	}
	return false
}

func makeLinkPrefix(r *http.Request) string {
	if r.TLS == nil {
		return "http://" + r.Host
	}
	return "https://" + r.Host
}
