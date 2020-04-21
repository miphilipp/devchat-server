package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	core "github.com/miphilipp/devchat-server/internal"
)

var apiErrorToStatusCodeMap = map[int]int{
	1000: http.StatusInternalServerError,
	1001: http.StatusNotFound,
	1002: http.StatusBadRequest,
	1003: http.StatusBadRequest,
	1005: http.StatusForbidden,
	1006: http.StatusForbidden,
	1007: http.StatusNotFound,
	1008: http.StatusInternalServerError,
	1009: http.StatusNotFound,
	1010: http.StatusBadRequest,
	1011: http.StatusBadRequest,
	1012: http.StatusBadRequest,
	1013: http.StatusNotFound,
	1014: http.StatusUnsupportedMediaType,
	1015: http.StatusBadRequest,
	1016: http.StatusBadRequest,
	1017: http.StatusBadRequest,
	1018: http.StatusBadRequest,
	1019: http.StatusTooManyRequests,
	1020: http.StatusUnauthorized,
	1021: http.StatusBadRequest,
	1022: http.StatusUnauthorized,
	1023: http.StatusBadRequest,
}

// SetupRestHandlers registers all the  REST routes
func (s Webserver) SetupRestHandlers() {
	s.router.HandleFunc("/user", func(writer http.ResponseWriter, request *http.Request) {
		s.registerUser(writer, request)
	}).Methods(http.MethodPost)

	s.router.HandleFunc("/login", func(writer http.ResponseWriter, request *http.Request) {
		s.login(writer, request)
	}).Methods(http.MethodPost)

	s.router.HandleFunc("/logout", func(writer http.ResponseWriter, request *http.Request) {
		s.logout(writer, request)
	}).Methods(http.MethodGet)

	s.router.HandleFunc("/user/confirm", func(writer http.ResponseWriter, request *http.Request) {
		s.confirmAccount(writer, request)
	}).Methods(http.MethodPatch)

	s.router.HandleFunc("/sendpasswordreset", func(writer http.ResponseWriter, request *http.Request) {
		s.sendPasswordReset(writer, request)
	}).Methods(http.MethodPost)

	s.router.HandleFunc("/passwordreset", func(writer http.ResponseWriter, request *http.Request) {
		s.resetPassword(writer, request)
	}).Methods(http.MethodPost)

	api := s.router.PathPrefix("/api/v1").Subrouter()
	api.Use(s.generateAuthenticateSession())

	media := s.router.PathPrefix("/media").Subrouter()
	media.Use(s.generateMediaAuthenticationMiddleware())
	media.HandleFunc("/user/{userid}/avatar", func(writer http.ResponseWriter, request *http.Request) {
		s.serveUserAvatar(writer, request)
	}).Methods(http.MethodGet)
	media.HandleFunc("/conversation/{conversationID}/message/{messageID}/media", func(writer http.ResponseWriter, request *http.Request) {
		s.serveMediaMessageRessource(writer, request)
	}).Methods(http.MethodGet)

	api.HandleFunc("/mediatoken", func(writer http.ResponseWriter, request *http.Request) {
		s.getMediaToken(writer, request)
	}).Methods(http.MethodGet)

	api.HandleFunc("/conversation", func(writer http.ResponseWriter, request *http.Request) {
		s.getConversation(writer, request)
	}).Methods(http.MethodGet)

	api.HandleFunc("/conversation", func(writer http.ResponseWriter, request *http.Request) {
		s.postConversation(writer, request)
	}).Methods(http.MethodPost)

	api.HandleFunc("/conversation/{id}", func(writer http.ResponseWriter, request *http.Request) {
		s.deleteConversation(writer, request)
	}).Methods(http.MethodDelete)

	api.HandleFunc("/conversation/{id}", func(writer http.ResponseWriter, request *http.Request) {
		s.patchConversation(writer, request)
	}).Methods(http.MethodPatch)

	api.HandleFunc("/conversation/{id}/users", func(writer http.ResponseWriter, request *http.Request) {
		s.getMembersOfConversation(writer, request)
	}).Methods(http.MethodGet)

	api.HandleFunc("/conversation/{conversationID}/users/{userID}", func(writer http.ResponseWriter, request *http.Request) {
		s.deleteUserFromConversation(writer, request)
	}).Methods(http.MethodDelete)

	api.HandleFunc("/conversation/{conversationID}/users/{userID}", func(writer http.ResponseWriter, request *http.Request) {
		s.patchAdminStatus(writer, request)
	}).Methods(http.MethodPatch)

	api.HandleFunc("/invitation", func(writer http.ResponseWriter, request *http.Request) {
		s.getInvitations(writer, request)
	}).Methods(http.MethodGet)

	api.HandleFunc("/invitation", func(writer http.ResponseWriter, request *http.Request) {
		s.postInvitation(writer, request)
	}).Methods(http.MethodPost)

	api.HandleFunc("/invitation", func(writer http.ResponseWriter, request *http.Request) {
		s.patchInvitation(writer, request)
	}).Methods(http.MethodPatch)

	api.HandleFunc("/invitation", func(writer http.ResponseWriter, request *http.Request) {
		s.deleteInvitation(writer, request)
	}).Methods(http.MethodDelete)

	api.HandleFunc("/user/avatar", func(writer http.ResponseWriter, request *http.Request) {
		s.postNewAvatar(writer, request)
	}).Methods(http.MethodPost)

	api.HandleFunc("/user/avatar", func(writer http.ResponseWriter, request *http.Request) {
		s.deleteAvatar(writer, request)
	}).Methods(http.MethodDelete)

	api.HandleFunc("/user", func(writer http.ResponseWriter, request *http.Request) {
		s.getProfile(writer, request)
	}).Methods(http.MethodGet)

	api.HandleFunc("/user", func(writer http.ResponseWriter, request *http.Request) {
		s.deleteUserAccount(writer, request)
	}).Methods(http.MethodDelete)

	api.HandleFunc("/user/password", func(writer http.ResponseWriter, request *http.Request) {
		s.patchPassword(writer, request)
	}).Methods(http.MethodPatch)

	api.HandleFunc("/users", func(writer http.ResponseWriter, request *http.Request) {
		s.getUsers(writer, request)
	}).Queries("prefix", "{prefix}").Methods(http.MethodGet)

	api.HandleFunc("/conversation/{id}/messages", func(writer http.ResponseWriter, request *http.Request) {
		s.getMessages(writer, request)
	}).Methods(http.MethodGet)

	api.HandleFunc("/conversation/{id}/messages/{messageID}/code", func(writer http.ResponseWriter, request *http.Request) {
		s.getCodeOfMessage(writer, request)
	}).Methods(http.MethodGet)

	api.HandleFunc("/conversation/{id}/messages/{messageID}", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Println("getMessage")
		s.getMessage(writer, request)
	}).Methods(http.MethodGet)

	api.HandleFunc("/programmingLanguages", func(writer http.ResponseWriter, request *http.Request) {
		s.getProgrammingLanguages(writer, request)
	}).Methods(http.MethodGet)

	api.HandleFunc("/websocket", func(writer http.ResponseWriter, request *http.Request) {
		userContext := request.Context().Value("UserID").(int)
		s.socket.StartWebsocket(writer, request, userContext)
	})
}

func checkForAPIError(err error, writer http.ResponseWriter) bool {
	err = core.UnwrapDatabaseError(err)
	if e, ok := err.(core.ApiError); ok {
		if errorCode, ok := apiErrorToStatusCodeMap[e.Code]; ok {
			writeJSONError(writer, e, errorCode)
		} else {
			writeJSONError(writer, e, http.StatusInternalServerError)
		}
		return true
	}
	return false
}

func writeJSONError(writer http.ResponseWriter, err error, statusCode int) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	json.NewEncoder(writer).Encode(err)
}
