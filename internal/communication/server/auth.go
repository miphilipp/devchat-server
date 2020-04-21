package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-kit/kit/log/level"
	"github.com/golang/gddo/httputil/header"
	core "github.com/miphilipp/devchat-server/internal"
	"github.com/miphilipp/devchat-server/internal/communication/session"
)

const accessTokenName = "access_token"

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

func (s *Webserver) logout(writer http.ResponseWriter, request *http.Request) {
	tokenString, err := getTokenFromRequest(request)
	if err != nil {
		checkForAPIError(core.ErrAuthFailed, writer)
		return
	}

	err = s.session.InvlidateToken(tokenString)
	if err != nil {
		level.Error(s.logger).Log("handler", "logout", "err", err)
		http.Error(writer, "Forbidden", http.StatusInternalServerError)
		return
	}
	writer.WriteHeader(http.StatusOK)
}

func getTokenFromRequest(request *http.Request) (string, error) {
	tokenString := request.Header.Get("Authorization")
	tokenString = strings.Replace(tokenString, "Bearer ", "", 1)

	if tokenString == "" {
		tokenCookie, err := request.Cookie(accessTokenName)
		if err != nil {
			return "", err
		}
		tokenString = tokenCookie.Value
	}
	return tokenString, nil
}

func (s *Webserver) generateAuthenticateSession() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			tokenString, err := getTokenFromRequest(request)
			if err != nil {
				checkForAPIError(core.ErrAuthFailed, writer)
				return
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

func (s *Webserver) login(writer http.ResponseWriter, request *http.Request) {
	if request.Header.Get("Content-Type") != "" {
		value, _ := header.ParseValueAndParams(request.Header, "Content-Type")
		if value != "application/json" {
			writeJSONError(writer, core.ErrRequireJSON, http.StatusUnsupportedMediaType)
			return
		}
	}

	loginData := struct {
		Password string
		Username string
	}{}
	err := json.NewDecoder(request.Body).Decode(&loginData)
	if err != nil {
		apiErrorJSON := core.NewJSONFormatError(err.Error())
		writeJSONError(writer, apiErrorJSON, http.StatusBadRequest)
		return
	}

	res, err := s.userService.AuthenticateUser(loginData.Username, loginData.Password)
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		}
		return
	}

	reply := struct {
		Success bool `json:"success"`
	}{}

	if res != -1 {
		token, err := s.session.GetToken(loginData.Username)
		if err != nil {
			level.Error(s.logger).Log("Handler", "login", "err", err)
			http.Error(writer, "Error", http.StatusInternalServerError)
			return
		}

		reply.Success = true
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Authorization", "Bearer "+token)
		cookie := http.Cookie{
			Name:     accessTokenName,
			Value:    token,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			Expires:  time.Now().UTC().Add(7 * 24 * time.Hour),
		}
		http.SetCookie(writer, &cookie)
		writer.WriteHeader(http.StatusOK)
		json.NewEncoder(writer).Encode(reply)
	} else {
		reply.Success = false

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		json.NewEncoder(writer).Encode(reply)
		return
	}
}
