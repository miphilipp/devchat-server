package server

import (
	"encoding/json"
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
func (s *Webserver) SetupRestHandlers() {
	s.router.HandleFunc("/user", func(writer http.ResponseWriter, request *http.Request) {
		err := s.registerUser(writer, request)
		if err != nil {
			sendAPIError(err, writer)
		}
	}).Methods(http.MethodPost)

	s.router.HandleFunc("/login", func(writer http.ResponseWriter, request *http.Request) {
		err := s.login(writer, request)
		if err != nil {
			sendAPIError(err, writer)
		}
	}).Methods(http.MethodPost)

	s.router.HandleFunc("/logout", func(writer http.ResponseWriter, request *http.Request) {
		s.logout(writer, request)
	}).Methods(http.MethodGet)

	s.router.HandleFunc("/user/confirm", func(writer http.ResponseWriter, request *http.Request) {
		err := s.confirmAccount(writer, request)
		if err != nil {
			sendAPIError(err, writer)
		}
	}).Methods(http.MethodPatch)

	s.router.HandleFunc("/sendpasswordreset", func(writer http.ResponseWriter, request *http.Request) {
		err := s.sendPasswordReset(writer, request)
		if err != nil {
			sendAPIError(err, writer)
		}
	}).Methods(http.MethodPost)

	s.router.HandleFunc("/passwordreset", func(writer http.ResponseWriter, request *http.Request) {
		err := s.resetPassword(writer, request)
		if err != nil {
			sendAPIError(err, writer)
		}
	}).Methods(http.MethodPost)

	api := s.router.PathPrefix("/api/v1").Subrouter()
	api.Use(s.generateAuthenticateSession())
	media := s.router.PathPrefix("/media").Subrouter()
	media.Use(s.generateAuthenticateSession())
	media.HandleFunc("/user/{userid:[0-9]+}/avatar", func(writer http.ResponseWriter, request *http.Request) {
		err := s.serveUserAvatar(writer, request)
		if err != nil {
			sendAPIError(err, writer)
		}
	}).Methods(http.MethodGet)

	media.HandleFunc("/conversation/{conversationID:[0-9]+}/{fileName}",
		func(writer http.ResponseWriter, request *http.Request) {
			err := s.serveMediaMessageRessource(writer, request)
			if err != nil {
				sendAPIError(err, writer)
			}
		}).Methods(http.MethodGet)

	api.HandleFunc("/conversation", func(writer http.ResponseWriter, request *http.Request) {
		err := s.getConversation(writer, request)
		if err != nil {
			sendAPIError(err, writer)
		}
	}).Methods(http.MethodGet)

	api.HandleFunc("/conversation", func(writer http.ResponseWriter, request *http.Request) {
		err := s.postConversation(writer, request)
		if err != nil {
			sendAPIError(err, writer)
		}
	}).Methods(http.MethodPost)

	api.HandleFunc("/conversation/{id:[0-9]+}", func(writer http.ResponseWriter, request *http.Request) {
		err := s.deleteConversation(writer, request)
		if err != nil {
			sendAPIError(err, writer)
		}
	}).Methods(http.MethodDelete)

	api.HandleFunc("/conversation/{id:[0-9]+}", func(writer http.ResponseWriter, request *http.Request) {
		err := s.patchConversation(writer, request)
		if err != nil {
			sendAPIError(err, writer)
		}
	}).Methods(http.MethodPatch)

	api.HandleFunc("/conversation/{id:[0-9]+}/users", func(writer http.ResponseWriter, request *http.Request) {
		err := s.getMembersOfConversation(writer, request)
		if err != nil {
			sendAPIError(err, writer)
		}
	}).Methods(http.MethodGet)

	api.HandleFunc("/conversation/{id:[0-9]+}/message/{messageID:[0-9]+}/upload",
		func(writer http.ResponseWriter, request *http.Request) {
			err := s.uploadMedia(writer, request)
			if err != nil {
				sendAPIError(err, writer)
			}
		}).Methods(http.MethodPatch)

	api.HandleFunc("/conversation/{conversationID:[0-9]+}/users/{userID:[0-9]+}",
		func(writer http.ResponseWriter, request *http.Request) {
			err := s.deleteUserFromConversation(writer, request)
			if err != nil {
				sendAPIError(err, writer)
			}
		}).Methods(http.MethodDelete)

	api.HandleFunc("/conversation/{conversationID:[0-9]+}/users/{userID}",
		func(writer http.ResponseWriter, request *http.Request) {
			err := s.patchAdminStatus(writer, request)
			if err != nil {
				sendAPIError(err, writer)
			}
		}).Methods(http.MethodPatch)

	api.HandleFunc("/invitation", func(writer http.ResponseWriter, request *http.Request) {
		err := s.getInvitations(writer, request)
		if err != nil {
			sendAPIError(err, writer)
		}
	}).Methods(http.MethodGet)

	api.HandleFunc("/invitation", func(writer http.ResponseWriter, request *http.Request) {
		err := s.postInvitation(writer, request)
		if err != nil {
			sendAPIError(err, writer)
		}
	}).Methods(http.MethodPost)

	api.HandleFunc("/invitation", func(writer http.ResponseWriter, request *http.Request) {
		err := s.patchInvitation(writer, request)
		if err != nil {
			sendAPIError(err, writer)
		}
	}).Methods(http.MethodPatch)

	api.HandleFunc("/invitation", func(writer http.ResponseWriter, request *http.Request) {
		err := s.deleteInvitation(writer, request)
		if err != nil {
			sendAPIError(err, writer)
		}
	}).Methods(http.MethodDelete)

	api.HandleFunc("/user/avatar", func(writer http.ResponseWriter, request *http.Request) {
		err := s.postNewAvatar(writer, request)
		if err != nil {
			sendAPIError(err, writer)
		}
	}).Methods(http.MethodPost)

	api.HandleFunc("/user/avatar", func(writer http.ResponseWriter, request *http.Request) {
		err := s.deleteAvatar(writer, request)
		if err != nil {
			sendAPIError(err, writer)
		}
	}).Methods(http.MethodDelete)

	api.HandleFunc("/user", func(writer http.ResponseWriter, request *http.Request) {
		err := s.getProfile(writer, request)
		if err != nil {
			sendAPIError(err, writer)
		}
	}).Methods(http.MethodGet)

	api.HandleFunc("/user", func(writer http.ResponseWriter, request *http.Request) {
		err := s.deleteUserAccount(writer, request)
		if err != nil {
			sendAPIError(err, writer)
		}
	}).Methods(http.MethodDelete)

	api.HandleFunc("/user/password", func(writer http.ResponseWriter, request *http.Request) {
		err := s.patchPassword(writer, request)
		if err != nil {
			sendAPIError(err, writer)
		}
	}).Methods(http.MethodPatch)

	api.HandleFunc("/users", func(writer http.ResponseWriter, request *http.Request) {
		err := s.getUsers(writer, request)
		if err != nil {
			sendAPIError(err, writer)
		}
	}).Queries("prefix", "{prefix}").Methods(http.MethodGet)

	api.HandleFunc("/conversation/{id:[0-9]+}/messages", func(writer http.ResponseWriter, request *http.Request) {
		err := s.getMessages(writer, request)
		if err != nil {
			sendAPIError(err, writer)
		}
	}).Methods(http.MethodGet)

	api.HandleFunc("/conversation/{id:[0-9]+}/messages/{messageID:[0-9]+}/code",
		func(writer http.ResponseWriter, request *http.Request) {
			err := s.getCodeOfMessage(writer, request)
			if err != nil {
				sendAPIError(err, writer)
			}
		}).Methods(http.MethodGet)

	api.HandleFunc("/conversation/{id:[0-9]+}/messages/{messageID:[0-9]+}",
		func(writer http.ResponseWriter, request *http.Request) {
			err := s.getMessage(writer, request)
			if err != nil {
				sendAPIError(err, writer)
			}
		}).Methods(http.MethodGet)

	api.HandleFunc("/programmingLanguages", func(writer http.ResponseWriter, request *http.Request) {
		err := s.getProgrammingLanguages(writer, request)
		if err != nil {
			sendAPIError(err, writer)
		}
	}).Methods(http.MethodGet)

	api.HandleFunc("/websocket", func(writer http.ResponseWriter, request *http.Request) {
		userContext := request.Context().Value("UserID").(int)
		err := s.socket.StartWebsocket(writer, request, userContext)
		if err != nil {
			sendAPIError(err, writer)
		}
	})
}

func sendAPIError(err error, writer http.ResponseWriter) {
	err = core.UnwrapDatabaseError(err)
	if e, ok := err.(core.ApiError); ok {
		if errorCode, ok := apiErrorToStatusCodeMap[e.Code]; ok {
			writeJSONError(writer, e, errorCode)
		} else {
			writeJSONError(writer, e, http.StatusInternalServerError)
		}
		return
	}

	writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
}

func writeJSONError(writer http.ResponseWriter, err error, statusCode int) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	json.NewEncoder(writer).Encode(err)
}
