package server

import (
	"bufio"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/mux"
	core "github.com/miphilipp/devchat-server/internal"
	"golang.org/x/text/language"
)

type userSearchResponse struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func (s *Webserver) serveUserAvatar(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	userID, err := strconv.Atoi(vars["userid"])
	if err != nil {
		level.Error(s.logger).Log("Handler", "serveUserAvatar", "err", err)
		apiErrorPath := core.NewPathFormatError(err.Error())
		writeJSONError(writer, apiErrorPath, http.StatusBadRequest)
		return
	}

	nodefaultStr := request.FormValue("nodefault")
	nodefault, err := strconv.ParseBool(nodefaultStr)
	if err != nil {
		nodefault = false
	}

	avatarPath, modTime, err := s.userService.GetAvatar(userID, s.config.AvatarFolder, nodefault)
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		}
		return
	}

	modTimeBin, err := modTime.MarshalBinary()
	if err != nil {
		level.Error(s.logger).Log("Handler", "serveUserAvatar", "err", err)
		writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		return
	}

	hash := fmt.Sprintf("\"%x\"", md5.Sum(modTimeBin))
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("ETag", hash)
	http.ServeFile(writer, request, avatarPath)
}

func (s *Webserver) registerUser(writer http.ResponseWriter, request *http.Request) {

	requestBody := struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}{"-", "-", "-"}

	err := json.NewDecoder(request.Body).Decode(&requestBody)
	if err != nil {
		level.Error(s.logger).Log("Handler", "registerUser", "err", err)
		apiErrorJSON := core.NewJSONFormatError(err.Error())
		writeJSONError(writer, apiErrorJSON, http.StatusBadRequest)
		return
	}

	if requestBody.Username == "-" {
		apiErrorJSON := core.NewJSONFormatError("username potentially missing")
		writeJSONError(writer, apiErrorJSON, http.StatusBadRequest)
		return
	}

	if requestBody.Password == "-" {
		apiErrorJSON := core.NewJSONFormatError("password potentially missing")
		writeJSONError(writer, apiErrorJSON, http.StatusBadRequest)
		return
	}

	if requestBody.Email == "-" {
		apiErrorJSON := core.NewJSONFormatError("email potentially missing")
		writeJSONError(writer, apiErrorJSON, http.StatusBadRequest)
		return
	}

	user := core.User{
		Email: requestBody.Email,
		Name:  requestBody.Username,
	}
	err = s.userService.CreateAccount(user, requestBody.Password, s.config.RootURL)
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		}
		return
	}

	writer.WriteHeader(http.StatusOK)
}

func (s *Webserver) confirmAccount(writer http.ResponseWriter, request *http.Request) {
	requestBody := struct {
		Token string `json:"token"`
	}{"-"}

	err := json.NewDecoder(request.Body).Decode(&requestBody)
	if err != nil {
		level.Error(s.logger).Log("Handler", "confirmAccount", "err", err)
		apiErrorJSON := core.NewJSONFormatError(err.Error())
		writeJSONError(writer, apiErrorJSON, http.StatusBadRequest)
		return
	}

	if requestBody.Token == "-" {
		apiErrorJSON := core.NewJSONFormatError("token potentially missing")
		writeJSONError(writer, apiErrorJSON, http.StatusBadRequest)
		return
	}

	username, err := s.userService.ConfirmAccount(requestBody.Token)
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		}
		return
	}

	reply := struct {
		Username string `json:"username"`
	}{username}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(reply)
}

func (s *Webserver) getProfile(writer http.ResponseWriter, request *http.Request) {
	userID := request.Context().Value("UserID").(int)
	self, err := s.userService.GetUserForID(userID)
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		}
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(self)
}

func (s *Webserver) getUsers(writer http.ResponseWriter, request *http.Request) {
	prefix := request.FormValue("prefix")
	users, err := s.userService.SearchUsers(prefix)
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		}
		return
	}

	response := make([]userSearchResponse, 0, len(users))
	for _, user := range users {
		response = append(response, userSearchResponse{
			ID:   user.ID,
			Name: user.Name,
		})
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(response)
}

func (s *Webserver) sendPasswordReset(writer http.ResponseWriter, request *http.Request) {
	language := request.Context().Value("Language").(language.Tag)
	requestBody := struct {
		Email string `json:"email"`
	}{"-"}

	err := json.NewDecoder(request.Body).Decode(&requestBody)
	if err != nil {
		level.Error(s.logger).Log("Handler", "sendPasswordRecovery", "err", err)
		apiErrorJSON := core.NewJSONFormatError(err.Error())
		writeJSONError(writer, apiErrorJSON, http.StatusBadRequest)
		return
	}

	if requestBody.Email == "-" {
		apiErrorJSON := core.NewJSONFormatError("email potentially missing")
		writeJSONError(writer, apiErrorJSON, http.StatusBadRequest)
		return
	}

	err = s.userService.SendPasswordResetMail(requestBody.Email, s.config.RootURL, language.String())
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		}
		return
	}

	writer.WriteHeader(http.StatusOK)
}

func (s *Webserver) patchPassword(writer http.ResponseWriter, request *http.Request) {
	userCtx := request.Context().Value("UserID").(int)
	userCtxName := request.Context().Value("username").(string)

	requestBody := struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}{"-", "-"}
	err := json.NewDecoder(request.Body).Decode(&requestBody)
	if err != nil {
		level.Error(s.logger).Log("Handler", "patchPassword", "err", err)
		apiErrorJSON := core.NewJSONFormatError(err.Error())
		writeJSONError(writer, apiErrorJSON, http.StatusBadRequest)
		return
	}

	if requestBody.OldPassword == "-" {
		apiErrorJSON := core.NewJSONFormatError("oldPassword potentially missing")
		writeJSONError(writer, apiErrorJSON, http.StatusBadRequest)
		return
	}

	if requestBody.NewPassword == "-" {
		apiErrorJSON := core.NewJSONFormatError("newPassword potentially missing")
		writeJSONError(writer, apiErrorJSON, http.StatusBadRequest)
		return
	}

	err = s.userService.ChangePassword(userCtx, requestBody.OldPassword, requestBody.NewPassword)
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		}
		return
	}

	s.session.InvlidateAllTokens(userCtxName)
	writer.WriteHeader(http.StatusOK)
}

func (s *Webserver) resetPassword(writer http.ResponseWriter, request *http.Request) {
	requestBody := struct {
		RecoveryUUID string `json:"recoveryUUID"`
		Password     string `json:"password"`
	}{"-", "-"}
	err := json.NewDecoder(request.Body).Decode(&requestBody)
	if err != nil {
		level.Error(s.logger).Log("Handler", "recoverPassword", "err", err)
		apiErrorJSON := core.NewJSONFormatError(err.Error())
		writeJSONError(writer, apiErrorJSON, http.StatusBadRequest)
		return
	}

	if requestBody.Password == "-" {
		apiErrorJSON := core.NewJSONFormatError("password potentially missing")
		writeJSONError(writer, apiErrorJSON, http.StatusBadRequest)
		return
	}

	if requestBody.RecoveryUUID == "-" {
		apiErrorJSON := core.NewJSONFormatError("recoveryUUID potentially missing")
		writeJSONError(writer, apiErrorJSON, http.StatusBadRequest)
		return
	}

	username, err := s.userService.ResetPassword(requestBody.RecoveryUUID, requestBody.Password)
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		}
		return
	}

	s.session.InvlidateAllTokens(username)

	reply := struct {
		Username string `json:"username"`
	}{username}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(reply)
}

func (s *Webserver) deleteUserAccount(writer http.ResponseWriter, request *http.Request) {
	userID := request.Context().Value("UserID").(int)
	userCtxName := request.Context().Value("username").(string)

	err := s.userService.DeleteAccount(userID)
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		}
		return
	}

	s.userService.DeleteAvatar(s.config.AvatarFolder, userID)

	s.socket.DisconnectClient(userID)
	s.session.InvlidateAllTokens(userCtxName)
	writer.WriteHeader(http.StatusOK)
}

func (s *Webserver) postNewAvatar(writer http.ResponseWriter, request *http.Request) {
	userID := request.Context().Value("UserID").(int)
	err := request.ParseMultipartForm(2 << 20)
	if err != nil {
		level.Error(s.logger).Log("Handler", "postNewAvatar", "err", err)
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	file, header, err := request.FormFile("avatar")
	if err != nil {
		level.Error(s.logger).Log("Handler", "postNewAvatar", "err", err)
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	buf := bufio.NewReaderSize(file, int(header.Size))
	peakAmount := min(header.Size, 512)
	sniff, err := buf.Peek(int(peakAmount))
	if err != nil {
		level.Error(s.logger).Log("Handler", "postNewAvatar", "err", err)
		writeJSONError(writer, core.ErrUnknownError, http.StatusBadRequest)
		return
	}

	contentType := http.DetectContentType(sniff)
	buffer := make([]byte, header.Size)
	buf.Read(buffer)

	err = s.userService.SaveAvatar(userID, s.config.AvatarFolder, contentType, buffer)
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		}
		return
	}

	writer.WriteHeader(http.StatusOK)
}

func (s *Webserver) deleteAvatar(writer http.ResponseWriter, request *http.Request) {
	userID := request.Context().Value("UserID").(int)
	err := s.userService.DeleteAvatar(s.config.AvatarFolder, userID)
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		}
		return
	}
	writer.WriteHeader(http.StatusOK)
}
