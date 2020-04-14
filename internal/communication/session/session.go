package session

import (
	"time"
	//"fmt"
	"errors"
	jwt "github.com/dgrijalva/jwt-go"
	core "github.com/miphilipp/devchat-server/internal"
)

const (
	ErrInvalidToken = "Invalid token"
)

type SessionManager struct {
	persistance SessionPersistance
	ttl time.Duration
}

func NewSessionManager(persistance SessionPersistance) *SessionManager {
	return &SessionManager{
		persistance: persistance,
		ttl: 7 * 24 * time.Hour,
	}
}

var secret = []byte("supersecret") //TODO: Austauschen

func (s *SessionManager) ValidateToken(token string) (string, error) {

	claims, err := verifyToken(token)
	if err != nil && claims == nil {
		return "", core.ErrInvalidToken
	}

	tokenIsValid := err == nil

	userName := claims.(jwt.MapClaims)["name"].(string)
	isBlackListed, err := s.persistance.IsBlackListed(userName, token)
	if err != nil {
		return "", err
	}

	if isBlackListed {
		return "", core.ErrInvalidToken
	}

	if !tokenIsValid {
		return "", core.ErrInvalidToken
	}

	return claims.(jwt.MapClaims)["name"].(string), nil
}

func (s *SessionManager) GetToken(name string) (string, error) {
	exp := time.Now().UTC().Add(s.ttl).Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"name": name,
		"exp": exp,
	})
	tokenString, err := token.SignedString(secret)

	err = s.persistance.Store(name, tokenString, exp)
	if err != nil {
		return "", err
	}

	return tokenString, err
}

func (s *SessionManager) InvlidateToken(token string) error {
	claims, err := verifyToken(token)
	if err != nil && claims == nil {
		return errors.New(ErrInvalidToken)
	}

	_, ok := claims.(jwt.MapClaims)["name"]
	if !ok {
		return errors.New(ErrInvalidToken)
	}

	_, ok = claims.(jwt.MapClaims)["exp"]
	if !ok {
		return errors.New(ErrInvalidToken)
	}

	userName := claims.(jwt.MapClaims)["name"].(string)
	exp := claims.(jwt.MapClaims)["exp"].(float64)

	err = s.persistance.BlackList(userName, token, exp)
	if err != nil {
		return err
	}

	return nil
}

func (s *SessionManager) InvlidateAllTokens(username string) error {
	return s.persistance.BlackListAll(username)
}

func verifyToken(tokenString string) (jwt.Claims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		return token.Claims, err
	}

	return token.Claims, err
}
