package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
)

var jwtKey = []byte(os.Getenv("SSE_APP_KEY"))

type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

type User struct {
	Name     string `json:"username"`
	Password string `json:"password"`
	Token    string
}

//Jwt handler to issue the token
func TokenHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	var u User
	err := json.NewDecoder(r.Body).Decode(&u)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//todo make it stronger
	if false == checkAuth(u.Name, u.Password) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"error":"Invalid creds"}`)
		return
	}

	expiresAt := expiresAt()
	claims := &Claims{
		Username: u.Name,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expiresAt.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, `{"error":"Token generation failed"}`)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:    "sse_token",
		Value:   tokenString,
		Expires: expiresAt,
	})

	tokenJson := fmt.Sprintf("{\"token\":\"%s\"}", tokenString)
	io.WriteString(w, tokenJson)
}

//handler wrapper for jwt auth
func AuthMiddleware(next http.Handler) http.Handler {
	if len(os.Getenv("SSE_APP_KEY")) == 0 {
		log.Fatal("Missing APP_KEY")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("sse_token")
		if err != nil {
			if err == http.ErrNoCookie { //no cookie, no token
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		cookieValue := c.Value
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(cookieValue, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})
		if err != nil {
			if err == jwt.ErrSignatureInvalid {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if !token.Valid {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r) //next should be the broker and its ServeHTTP function, in our case
	})

}

//todo make it stronger
func checkAuth(username string, password string) bool {
	return username == os.Getenv("SSE_USERNAME") && password == os.Getenv("SSE_PASSWORD")
}

//one hour ?
func expiresAt() time.Time {
	return time.Now().Add(time.Hour * time.Duration(1))
}
