package server

import (
	"net/http"
	"os"
	"bufio"
	//"fmt"
	"strconv"
	"encoding/json"

	"golang.org/x/text/language"
	"github.com/gorilla/mux"
	"github.com/go-kit/kit/log/level"
	core "github.com/miphilipp/devchat-server/internal"
)

type userSearchResponse struct {
	ID int `json:"id"`
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

	defaultAvatarPath := s.staticPath + "/../avatars/default.png"
	path := makeAvatarFileName(userID, s.staticPath)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if nodefault {
			http.Error(writer, "Not found", http.StatusNotFound)
			return
		}
		
		path = defaultAvatarPath
	} 

	http.ServeFile(writer, request, path)
}

func (s *Webserver) registerUser(writer http.ResponseWriter, request *http.Request) {

	requestBody := struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email	 string `json:"email"`
	}{ "-", "-", "-" }

	err := json.NewDecoder(request.Body).Decode(&requestBody)
	if err != nil {
		level.Error(s.logger).Log("Handler", "registerUser", "err", err)
		apiErrorJSON := core.NewJSONFormatError(err.Error())
		writeJSONError(writer, apiErrorJSON, http.StatusBadRequest)
		return
	}

	if requestBody.Username == "-" {
		apiErrorJSON := core.NewJSONFormatError("username missing")
		writeJSONError(writer, apiErrorJSON, http.StatusBadRequest)
		return
	}

	if requestBody.Password == "-" {
		apiErrorJSON := core.NewJSONFormatError("password missing")
		writeJSONError(writer, apiErrorJSON, http.StatusBadRequest)
		return
	}

	if requestBody.Email == "-" {
		apiErrorJSON := core.NewJSONFormatError("email missing")
		writeJSONError(writer, apiErrorJSON, http.StatusBadRequest)
		return
	}

	user := core.User{
		Email: requestBody.Email,
		Name: requestBody.Username,
	}
	err = s.userService.CreateAccount(user, requestBody.Password, makeLinkPrefix(request))
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
		Token	 string `json:"token"`
	}{ "-" }

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
	}{ username }

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
			ID: user.ID,
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
		Email	 string `json:"email"`
	}{ "-" }

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

	err = s.userService.SendPasswordResetMail(requestBody.Email, makeLinkPrefix(request), language.String())
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		} 
		return
	}

	writer.WriteHeader(http.StatusOK)
}

func (s *Webserver) patchPassword(writer http.ResponseWriter, request *http.Request)  {
	userCtx := request.Context().Value("UserID").(int)
	user, err := s.userService.GetUserForID(userCtx)
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		} 
		return
	}

	requestBody := struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}{"-", "-"}
	err = json.NewDecoder(request.Body).Decode(&requestBody)
	if err != nil {
		level.Error(s.logger).Log("Handler", "patchPassword", "err", err)
		apiErrorJSON := core.NewJSONFormatError(err.Error())
		writeJSONError(writer, apiErrorJSON, http.StatusBadRequest)
		return
	}

	if requestBody.OldPassword == "-" {
		apiErrorJSON := core.NewJSONFormatError("oldPassword missing")
		writeJSONError(writer, apiErrorJSON, http.StatusBadRequest)
		return
	}

	if requestBody.NewPassword == "-" {
		apiErrorJSON := core.NewJSONFormatError("newPassword missing")
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

	s.session.InvlidateAllTokens(user.Name)
	writer.WriteHeader(http.StatusOK)
}

func (s *Webserver) resetPassword(writer http.ResponseWriter, request *http.Request) {
	requestBody := struct {
		RecoveryUUID string `json:"recoveryUUID"`
		Password string `json:"password"`
	}{"-",  "-"}
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
	}{ username }

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(reply)
}

func (s *Webserver) deleteUserAccount(writer http.ResponseWriter, request *http.Request) {
	userID := request.Context().Value("UserID").(int)
	user, err := s.userService.GetUserForID(userID)
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		} 
		return
	}

	err = s.userService.DeleteAccount(userID)
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		} 
		return
	}

	s.userService.DeleteAvatar(makeAvatarFileName(userID, s.staticPath))

	s.socket.DisconnectClient(userID)
	s.session.InvlidateAllTokens(user.Name)
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
	sniff, err := buf.Peek(512)
	if err != nil {
		level.Error(s.logger).Log("Handler", "postNewAvatar", "err", err)
		writeJSONError(writer, core.ErrUnknownError, http.StatusBadRequest)
        return
	}

	contentType := http.DetectContentType(sniff)
	buffer := make([]byte, header.Size)
	buf.Read(buffer)

	err = s.userService.SaveAvatar(makeAvatarFileName(userID, s.staticPath), contentType, buffer)
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
	err := s.userService.DeleteAvatar(makeAvatarFileName(userID, s.staticPath))
	if err != nil {
		if !checkForAPIError(err, writer) {
			writeJSONError(writer, core.ErrUnknownError, http.StatusInternalServerError)
		} 
		return
	} 
	writer.WriteHeader(http.StatusOK)
}

func makeAvatarFileName(userID int, pathPrefix string) string {
	return pathPrefix + "/../avatars/" + strconv.Itoa(userID) + ".png"
}