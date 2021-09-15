package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/cliffordlab/amoss_services/capacity"
	jwt "github.com/dgrijalva/jwt-go"
)

const (
	post = "POST"
)

//AmossLoginRequest needs to be public so json can be accessible by other packages

//HandleReq wrapper for all requests to api
func HandleReq(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Stop here if its Preflighted OPTIONS request
		if r.Method == "OPTIONS" {
			if origin := r.Header.Get("Origin"); origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
				w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
				w.Header().Set("Access-Control-Allow-Headers", "Accept, Accept-Language, Content-Type, Authorization, Access-Control-Allow-Headers")
				w.Write([]byte(""))
			}
			return
		}

		before := time.Now().UnixNano() / 1000000
		w.Header().Add("Content-Type", "application/json; charset=UTF-8")
		h.ServeHTTP(w, r)
		after := time.Now().UnixNano() / 1000000
		//calculating the speed of entire request in millis
		log.Printf("%s request latency: %d\n", h, (after - before))
	})
}

func HandleReqWithBearerToken(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Stop here if its Preflighted OPTIONS request
		if r.Method == "OPTIONS" {
			if origin := r.Header.Get("Origin"); origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
				w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
				w.Header().Set("Access-Control-Allow-Headers", "Accept, Accept-Language, Content-Type, Authorization, Access-Control-Allow-Headers")
				w.Write([]byte(""))
			}
			return
		}

		before := time.Now().UnixNano() / 1000000
		w.Header().Add("Content-Type", "application/json; charset=UTF-8")

		headerValue := r.Header.Get("Authorization")
		splitHeaderValue := strings.Split(headerValue, " ")
		if splitHeaderValue[0] != "Bearer" {
			http.Error(w, "Invalid token type", http.StatusUnauthorized)
			return
		}
		marsToken := splitHeaderValue[1]
		token, err := jwt.ParseWithClaims(marsToken, &capacity.NonAdminClaims{}, func(token *jwt.Token) (interface{}, error) {
			// Make sure token's signature wasn't changed
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected siging method")
			}
			return []byte(capacity.JwtSecret), nil
		})
		if err != nil {
			log.Println("unable to parse with claims")
			http.Error(w, "Invalid token type", http.StatusUnauthorized)
			return
		}

		if claims, ok := token.Claims.(*capacity.NonAdminClaims); ok && token.Valid {
			if claims.Capacity == "coordinator" {
				h.ServeHTTP(w, r)
			} else {
				log.Println("token not valid")
				http.Error(w, `{"error": "Invalid token type"}`, http.StatusOK)
				return
			}
		} else {
			log.Println("token not valid")
			http.Error(w, `{"error": "Invalid token type"}`, http.StatusOK)
			return
		}

		after := time.Now().UnixNano() / 1000000
		//calculating the speed of entire request in millis
		log.Printf("%s request latency: %d\n", h, (after - before))
	})
}
