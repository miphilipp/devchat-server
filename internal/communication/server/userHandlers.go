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

func (s *Webserver) serveUserAvatar(writer http.ResponseWriter, request *http.Request) error {
	vars := mux.Vars(request)
	userID, err := strconv.Atoi(vars["userid"])
	if err != nil {
		level.Error(s.logger).Log("Handler", "serveUserAvatar", "err", err)
		return core.NewPathFormatError(err.Error())
	}

	nodefaultStr := request.FormValue("nodefault")
	nodefault, err := strconv.ParseBool(nodefaultStr)
	if err != nil {
		nodefault = false
	}

	avatarPath, modTime, err := s.userService.GetAvatar(userID, s.config.AvatarFolder, nodefault)
	if err != nil {
		return err
	}

	modTimeBin, err := modTime.MarshalBinary()
	if err != nil {
		level.Error(s.logger).Log("Handler", "serveUserAvatar", "err", err)
		return core.ErrUnknownError
	}

	hash := fmt.Sprintf("\"%x\"", md5.Sum(modTimeBin))
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("ETag", hash)
	http.ServeFile(writer, request, avatarPath)
	return nil
}

func (s *Webserver) registerUser(writer http.ResponseWriter, request *http.Request) error {

	requestBody := struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}{"-", "-", "-"}

	err := json.NewDecoder(request.Body).Decode(&requestBody)
	if err != nil {
		level.Error(s.logger).Log("Handler", "registerUser", "err", err)
		return core.NewJSONFormatError(err.Error())
	}

	if requestBody.Username == "-" {
		return core.NewJSONFormatError("username potentially missing")
	}

	if requestBody.Password == "-" {
		return core.NewJSONFormatError("password potentially missing")
	}

	if requestBody.Email == "-" {
		return core.NewJSONFormatError("email potentially missing")
	}

	user := core.User{
		Email: requestBody.Email,
		Name:  requestBody.Username,
	}
	err = s.userService.CreateAccount(user, requestBody.Password, s.config.RootURL)
	if err != nil {
		return err
	}

	writer.WriteHeader(http.StatusOK)
	return nil
}

func (s *Webserver) confirmAccount(writer http.ResponseWriter, request *http.Request) error {
	requestBody := struct {
		Token string `json:"token"`
	}{"-"}

	err := json.NewDecoder(request.Body).Decode(&requestBody)
	if err != nil {
		level.Error(s.logger).Log("Handler", "confirmAccount", "err", err)
		return core.NewJSONFormatError(err.Error())
	}

	if requestBody.Token == "-" {
		return core.NewJSONFormatError("token potentially missing")
	}

	username, err := s.userService.ConfirmAccount(requestBody.Token)
	if err != nil {
		return err
	}

	reply := struct {
		Username string `json:"username"`
	}{username}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(reply)
	return nil
}

func (s *Webserver) getProfile(writer http.ResponseWriter, request *http.Request) error {
	userID := request.Context().Value("UserID").(int)
	self, err := s.userService.GetUserForID(userID)
	if err != nil {
		return err
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(self)
	return nil
}

func (s *Webserver) getUsers(writer http.ResponseWriter, request *http.Request) error {
	prefix := request.FormValue("prefix")
	users, err := s.userService.SearchUsers(prefix)
	if err != nil {
		return err
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
	return nil
}

func (s *Webserver) sendPasswordReset(writer http.ResponseWriter, request *http.Request) error {
	language := request.Context().Value("Language").(language.Tag)
	requestBody := struct {
		Email string `json:"email"`
	}{"-"}

	err := json.NewDecoder(request.Body).Decode(&requestBody)
	if err != nil {
		level.Error(s.logger).Log("Handler", "sendPasswordRecovery", "err", err)
		return core.NewJSONFormatError(err.Error())
	}

	if requestBody.Email == "-" {
		return core.NewJSONFormatError("email potentially missing")
	}

	err = s.userService.SendPasswordResetMail(requestBody.Email, s.config.RootURL, language.String())
	if err != nil {
		return err
	}

	writer.WriteHeader(http.StatusOK)
	return nil
}

func (s *Webserver) patchPassword(writer http.ResponseWriter, request *http.Request) error {
	userCtx := request.Context().Value("UserID").(int)
	userCtxName := request.Context().Value("username").(string)

	requestBody := struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}{"-", "-"}
	err := json.NewDecoder(request.Body).Decode(&requestBody)
	if err != nil {
		level.Error(s.logger).Log("Handler", "patchPassword", "err", err)
		return core.NewJSONFormatError(err.Error())
	}

	if requestBody.OldPassword == "-" {
		return core.NewJSONFormatError("oldPassword potentially missing")
	}

	if requestBody.NewPassword == "-" {
		return core.NewJSONFormatError("newPassword potentially missing")
	}

	err = s.userService.ChangePassword(userCtx, requestBody.OldPassword, requestBody.NewPassword)
	if err != nil {
		return err
	}

	s.session.InvlidateAllTokens(userCtxName)
	writer.WriteHeader(http.StatusOK)
	return nil
}

func (s *Webserver) resetPassword(writer http.ResponseWriter, request *http.Request) error {
	requestBody := struct {
		RecoveryUUID string `json:"recoveryUUID"`
		Password     string `json:"password"`
	}{"-", "-"}
	err := json.NewDecoder(request.Body).Decode(&requestBody)
	if err != nil {
		level.Error(s.logger).Log("Handler", "recoverPassword", "err", err)
		return core.NewJSONFormatError(err.Error())
	}

	if requestBody.Password == "-" {
		return core.NewJSONFormatError("password potentially missing")
	}

	if requestBody.RecoveryUUID == "-" {
		return core.NewJSONFormatError("recoveryUUID potentially missing")
	}

	username, err := s.userService.ResetPassword(requestBody.RecoveryUUID, requestBody.Password)
	if err != nil {
		return err
	}

	s.session.InvlidateAllTokens(username)

	reply := struct {
		Username string `json:"username"`
	}{username}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(reply)
	return err
}

func (s *Webserver) deleteUserAccount(writer http.ResponseWriter, request *http.Request) error {
	userID := request.Context().Value("UserID").(int)
	userCtxName := request.Context().Value("username").(string)

	err := s.userService.DeleteAccount(userID)
	if err != nil {
		return err
	}

	s.userService.DeleteAvatar(s.config.AvatarFolder, userID)

	s.socket.DisconnectClient(userID)
	s.session.InvlidateAllTokens(userCtxName)
	writer.WriteHeader(http.StatusOK)
	return nil
}

func (s *Webserver) postNewAvatar(writer http.ResponseWriter, request *http.Request) error {
	userID := request.Context().Value("UserID").(int)
	err := request.ParseMultipartForm(2 << 20)
	if err != nil {
		level.Error(s.logger).Log("Handler", "postNewAvatar", "err", err)
		return err
	}

	file, header, err := request.FormFile("avatar")
	if err != nil {
		level.Error(s.logger).Log("Handler", "postNewAvatar", "err", err)
		return err
	}
	defer file.Close()

	buf := bufio.NewReaderSize(file, int(header.Size))
	peakAmount := min(header.Size, 512)
	sniff, err := buf.Peek(int(peakAmount))
	if err != nil {
		level.Error(s.logger).Log("Handler", "postNewAvatar", "err", err)
		return core.ErrUnknownError
	}

	contentType := http.DetectContentType(sniff)
	buffer := make([]byte, header.Size)
	buf.Read(buffer)

	err = s.userService.SaveAvatar(userID, s.config.AvatarFolder, contentType, buffer)
	if err != nil {
		return err
	}

	writer.WriteHeader(http.StatusOK)
	return nil
}

func (s *Webserver) deleteAvatar(writer http.ResponseWriter, request *http.Request) error {
	userID := request.Context().Value("UserID").(int)
	err := s.userService.DeleteAvatar(s.config.AvatarFolder, userID)
	if err != nil {
		return err
	}
	writer.WriteHeader(http.StatusOK)
	return nil
}
