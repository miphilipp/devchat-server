package session

import (
	"testing"

	jwt "github.com/dgrijalva/jwt-go"
)

func TestVerifyToken(t *testing.T) {
	secret := []byte("l36FjiSFBgXxyakipXUtxrs2V7wtGzHX")
	const expiredToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiSm9obiBEb2UiLCJleHAiOjEwMTYyMzkwMjJ9.h7WNmCGVtXwQxH1OJ4NoZRWd7ZqImlRNf_Ec_CI9dAg"
	const validToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiSm9obiBEb2UiLCJleHAiOjMwMTYyMzkwMjJ9.-dtxuB6fCXk58w_EpceYLy6aYJAdwyVg0KWl4DDJPQc"
	const badToken = "eyJhbGciOiJIUzddfI1NiJ9.eyJuYW1lIjoicGhpbGlwcCIsImV4cCIff4?6NjU4NDM3MjQ4Mn0.dK9QlWp6hr-eDObbRq53lCGQ1dwUaES_5zb2a_AOuOg"

	claims, err := VerifyToken(expiredToken, secret)
	if err == nil {
		t.Fail()
	}

	name := claims.(jwt.MapClaims)["name"].(string)
	if name != "John Doe" {
		t.Fail()
	}

	claims, err = VerifyToken(validToken, secret)
	if err != nil {
		t.Error(err.Error())
	}

	name = claims.(jwt.MapClaims)["name"].(string)
	if name != "John Doe" {
		t.Fail()
	}

	claims, err = VerifyToken(badToken, secret)
	if err == nil || claims != nil {
		t.Fail()
	}
}
