package capacity

import (
	"fmt"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

//JwtSecret will be set by main
var JwtSecret string
var AWSSESLambdaAPIKey string
var GarminSecret string
var GarminToken string
var CryptoKey string

//AdminClaims claims create token with administrative policy
type AdminClaims struct {
	ID       int64  `json:"participant_id"`
	Capacity string `json:"capacity"`
	jwt.StandardClaims
}

//NonAdminClaims claims create token with coordinator or patient policy
type NonAdminClaims struct {
	ID       int64  `json:"participant_id"`
	Capacity string `json:"capacity"`
	Study    string `json:"study"`
	jwt.StandardClaims
}

//CreateAccessToken for participants
func CreateAccessToken(capacity string, study string, ptID int64) string {
	//get signing key and expiration for token to apply to claims of jwt token
	signingKey := []byte(JwtSecret)
	var expireToken int64
	// Some studies need a longer expiration date. Default is 1 year.
	if study == "cfd-sleep-study-test" || study == "cfd-classroom-audio" || study == "cfd-sleep-study" {
		// Valid token 3 years
		fmt.Println("Token valid: 3 years")
		expireToken = time.Now().Add(time.Hour * 24 * 365 * 3).Unix()
	} else {
		// Default valid token is 1 year
		fmt.Println("Token valid: 1 year")
		expireToken = time.Now().Add(time.Hour * 24 * 365).Unix()
	}

	fmt.Printf("creating access token with %s capacity\n", capacity)

	var token *jwt.Token
	switch capacity {
	case "admin":
		claims := AdminClaims{
			ptID,
			capacity,
			jwt.StandardClaims{
				ExpiresAt: expireToken,
				Issuer:    "localhost:8080",
			},
		}
		token = jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	default:
		claims := NonAdminClaims{
			ptID,
			capacity,
			study,
			jwt.StandardClaims{
				ExpiresAt: expireToken,
				Issuer:    "localhost:8080",
			},
		}
		token = jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	}

	//create token with claims
	signedTokenString, _ := token.SignedString(signingKey)
	return signedTokenString
}
