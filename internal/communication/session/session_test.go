package session

import (
	"testing"
	jwt "github.com/dgrijalva/jwt-go"
)

func TestVerifyToken(t *testing.T) {
	const expiredToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoicGhpbGlwcCIsImV4cCI6MzMzMzN9.d8btnRKRO7dsv4pg12KC-ysWicVVXfd3pEw47Gl8orY"
	const validToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoicGhpbGlwcCIsImV4cCI6NjU4NDM3MjQ4Mn0.IfGHeFRFkpF_N5vF_fPSI6epuQJ1xCoKA14GU1imE94"
	const badToken = "eyJhbGciOiJIUzddfI1NiJ9.eyJuYW1lIjoicGhpbGlwcCIsImV4cCIff4?6NjU4NDM3MjQ4Mn0.dK9QlWp6hr-eDObbRq53lCGQ1dwUaES_5zb2a_AOuOg"

	claims, err := verifyToken(expiredToken)
	if err == nil {
		t.Fail()
	} else {
		t.Log(err.Error())
	}

	name := claims.(jwt.MapClaims)["name"].(string)
	if name != "philipp" {
		t.Fail()
	}

	claims, err = verifyToken(validToken)
	if err != nil {
		t.Fail()
	}

	name = claims.(jwt.MapClaims)["name"].(string)
	if name != "philipp" {
		t.Fail()
	}

	claims, err = verifyToken(badToken)
	if err == nil || claims != nil {
		t.Fail()
	}
}