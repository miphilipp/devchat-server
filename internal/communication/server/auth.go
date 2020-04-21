package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	core "github.com/miphilipp/devchat-server/internal"
	"github.com/miphilipp/devchat-server/internal/communication/session"
)

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

func (s *Webserver) generateAuthenticateSession() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			tokenString := request.Header.Get("Authorization")
			tokenString = strings.Replace(tokenString, "Bearer ", "", 1)

			tokenCookie, err := request.Cookie("access_token")
			if err != nil {
				checkForAPIError(core.ErrAuthFailed, writer)
				return
			}
			if tokenString == "" {
				tokenString = tokenCookie.Value
			}

			name, err := s.session.ValidateToken(tokenString)
			if err == nil {
				//fmt.Println("Authentifikation Erfolgreich")
				user, err := s.userService.GetUserForName(name)
				if err != nil {
					if !checkForAPIError(err, writer) {
						writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
					}
					return
				}
				copiedRequest := request.WithContext(context.WithValue(request.Context(), "UserID", user.ID))
				next.ServeHTTP(writer, copiedRequest)
			} else {
				checkForAPIError(core.ErrAuthFailed, writer)
			}
		})
	}
}
