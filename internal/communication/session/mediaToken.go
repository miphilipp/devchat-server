package session

import (
	"time"
	jwt "github.com/dgrijalva/jwt-go"
)

var mediasecret = []byte("mediasupersecret") //TODO: Austauschen

func GetMediaToken(ttl time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().UTC().Add(ttl).Unix(),
	})
	tokenString, err := token.SignedString(mediasecret)
	return tokenString, err
}

func VerifyMediaToken(tokenString string) (bool, error) {
	_, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return mediasecret, nil
	})
	if err != nil {
		return false, err
	}
	return true, nil
}
