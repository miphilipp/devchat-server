package session

import (
	"time"
	//"fmt"

	jwt "github.com/dgrijalva/jwt-go"
	core "github.com/miphilipp/devchat-server/internal"
)

type Manager struct {
	persistance Persistance
	ttl         time.Duration
	secret      []byte
}

func NewManager(persistance Persistance, secret []byte) *Manager {
	return &Manager{
		persistance: persistance,
		ttl:         7 * 24 * time.Hour,
		secret:      secret,
	}
}

// ValidateToken Checks if the token is valid and/or black listed.
func (s *Manager) ValidateToken(token string) (string, error) {

	claims, err := verifyToken(token, s.secret)
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

// GetToken creates and then returns a new token for the given user.
func (s *Manager) GetToken(name string) (string, error) {
	exp := time.Now().UTC().Add(s.ttl).Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"name": name,
		"exp":  exp,
	})
	tokenString, err := token.SignedString(s.secret)

	err = s.persistance.Store(name, tokenString, exp)
	if err != nil {
		return "", err
	}

	return tokenString, err
}

// InvlidateToken puts the token on the black list.
func (s *Manager) InvlidateToken(token string) error {
	claims, err := verifyToken(token, s.secret)
	if err != nil && claims == nil {
		return core.ErrInvalidToken
	}

	_, ok := claims.(jwt.MapClaims)["name"]
	if !ok {
		return core.ErrInvalidToken
	}

	_, ok = claims.(jwt.MapClaims)["exp"]
	if !ok {
		return core.ErrInvalidToken
	}

	userName := claims.(jwt.MapClaims)["name"].(string)
	exp := claims.(jwt.MapClaims)["exp"].(float64)

	err = s.persistance.BlackList(userName, token, exp)
	if err != nil {
		return err
	}

	return nil
}

// InvlidateAllTokens invalidates all tokens of a user by putting them
// on the black list.
func (s *Manager) InvlidateAllTokens(username string) error {
	return s.persistance.BlackListAll(username)
}

func verifyToken(tokenString string, secret []byte) (jwt.Claims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		return token.Claims, err
	}

	return token.Claims, err
}
