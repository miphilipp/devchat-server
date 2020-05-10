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
)

const accessTokenName = "access_token"

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
				user, err := s.userService.GetUserForName(name)
				if err != nil {
					if !checkForAPIError(err, writer) {
						writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
					}
					return
				}
				ctx := context.WithValue(request.Context(), "UserID", user.ID)
				ctx = context.WithValue(ctx, "username", user.Name)
				copiedRequest := request.WithContext(ctx)
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

	if res != -1 {
		token, err := s.session.GetSessionToken(loginData.Username)
		if err != nil {
			level.Error(s.logger).Log("Handler", "login", "err", err)
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
			return
		}

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
	} else {
		writeJSONError(writer, core.ErrAuthFailed, http.StatusUnauthorized)
	}
}
