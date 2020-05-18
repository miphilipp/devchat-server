package core

var (
	ErrFeatureDeactivated             = ApiError{1023, "This feature is currently not availiable"}
	ErrAccountNotConfirmed            = ApiError{1022, "Account hasn't been confirmed yet"}
	ErrAuthFailed                     = ApiError{1020, "User cannot be authenticated"}
	ErrRequestLimitExceeded           = ApiError{1019, "The client exceeded the request limit."}
	ErrInvalidFileType                = ApiError{1018, "File type not allowed"}
	ErrUnknownError                   = ApiError{1000, "An unexpected error occured."}
	ErrDataBase                       = ApiError{1008, "Some unexepected error occured within the datatbase."}
	ErrAccessDenied                   = ApiError{1006, "Access denied"}
	ErrRequireJSON                    = ApiError{1014, "JSON format required"}
	ErrAlreadyExists                  = ApiError{1021, "Ressource alread exists"}
	ErrUnsupportedMethod              = ApiError{1017, "Unsupported method"}
	ErrNothingChanged                 = ApiError{1007, "This action had no effect"}
	ErrLockedOut                      = ApiError{1005, "User Locked out"}
	ErrNoResultFromDB                 = ApiError{1004, "No rows returned"}
	ErrInvalidMessageType             = ApiError{1024, "This message type is not implemented."}
	ErrUserDoesNotExist               = ApiError{1001, "No such user"}
	ErrExpired                        = ApiError{1015, "Ressource expired"}
	ErrInvalidToken                   = ApiError{1016, "Invalid token"}
	ErrRessourceDoesNotExist          = ApiError{1013, "The requested ressource does not exist."}
	ErrConversationDoesNotExist       = ApiError{1009, "No such conversation"}
	ErrMessageTypeNotImplemented      = ApiError{1003, "This message type was not implemented"}
	ErrPasswordDoesNotMeetRequiremens = ApiError{1002, "The proposed password does not meet the requirements"}
)

// ApiError
type ApiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e ApiError) Error() string { return e.Message }

// RequestLimitExceededError represents an error when a client is making to many
// requests to the server.
// The RateLimit map should contain the following fields.
// All the units are in seconds.
// 	* retryAfter
// 	* remaining
// 	* limit
// 	* resetAfter
type RequestLimitExceededError struct {
	Err       ApiError
	RateLimit map[string]int
}

func (e RequestLimitExceededError) Error() string { return e.Err.Error() }

// DataBaseError combines core.ErrDataBase with a generic error return from a database.
type DataBaseError struct {
	APIError ApiError
	Err      error
}

// NewDataBaseError creates and returns a DataBaseError based on some error.
// Returns nil if the input is nil.
func NewDataBaseError(err error) error {
	if err == nil {
		return nil
	}
	return DataBaseError{
		APIError: ErrDataBase,
		Err:      err,
	}
}

// Unwrap retrieves the ErrDataBase instance inside any DataBaseError.
func (e DataBaseError) Unwrap() error { return e.APIError }

func (e DataBaseError) Error() string { return e.Err.Error() }

// UnwrapDatabaseError returns either the wrapped core.ErrDataBase if the passed in error
// is of type DataBaseError. Else it returns the passed in error as is.
func UnwrapDatabaseError(err error) error {
	if e, ok := err.(DataBaseError); ok {
		return e.Unwrap()
	}
	return err
}

// NewJSONFormatError creates an ApiError with code 1010. It is made to indicate
// invalid json payloads in requests.
func NewJSONFormatError(message string) ApiError {
	return ApiError{1010, message}
}

// NewInvalidValueError creates an ApiError with code 1012. It is made to indicate
// invalid values in requests.
func NewInvalidValueError(field string) ApiError {
	return ApiError{1012, field}
}

// NewPathFormatError creates an ApiError with code 1011. It is made to indicate
// invalid values in urls.
func NewPathFormatError(message string) ApiError {
	return ApiError{1011, message}
}

// NewRequestLimitExceededError creates an new RequestLimitExceededError
func NewRequestLimitExceededError(retryAfter, remaining, limit, resetAfter int) RequestLimitExceededError {
	return RequestLimitExceededError{
		Err: ErrRequestLimitExceeded,
		RateLimit: map[string]int{
			"retryAfter": retryAfter,
			"remaining":  remaining,
			"limit":      limit,
			"resetAfter": resetAfter,
		},
	}
}
