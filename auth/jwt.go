package auth

import (
	"context"
	"encoding/base64"
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
	Id       string `json:"id"`
	Token    string `json:"token"`
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

	tokenJson := fmt.Sprintf("{\"token\":\"%s\", \"id\": \"%s\"}", tokenString, u.Id)
	http.SetCookie(w, &http.Cookie{
		Name:    "sse_token",
		Value:   base64.StdEncoding.EncodeToString([]byte(tokenJson)),
		Expires: expiresAt,
	})

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

		//decode cookie
		cookieValue, err := base64.StdEncoding.DecodeString(c.Value)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, `{"error":"Cookie decoding failed"}`)
			return
		}

		//hydrate user with cookie datas
		var u User
		json.Unmarshal([]byte(cookieValue), &u)

		//check token's validity
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(u.Token, claims, func(token *jwt.Token) (interface{}, error) {
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

		//add user to request context
		ctx := r.Context()
		ctx = context.WithValue(ctx, "user", u)
		next.ServeHTTP(w, r.WithContext(ctx)) //next should be the broker and its ServeHTTP function, in our case
	})

}

//todo make it stronger
func checkAuth(username string, password string) bool {
	return username == os.Getenv("SSE_USERNAME") && password == os.Getenv("SSE_PASSWORD")
}

//one second ?
func expiresAt() time.Time {
	return time.Now().Add(time.Second * time.Duration(1))
}
