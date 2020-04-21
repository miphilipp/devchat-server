package session

import (
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

func GetMediaToken(ttl time.Duration, secret []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().UTC().Add(ttl).Unix(),
	})
	tokenString, err := token.SignedString(secret)
	return tokenString, err
}

func VerifyMediaToken(tokenString string, secret []byte) (bool, error) {
	_, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		return false, err
	}
	return true, nil
}
