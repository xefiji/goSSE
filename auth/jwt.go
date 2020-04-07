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
	"regexp"
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

func addCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", os.Getenv("ALLOWED_ORIGIN"))
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	(*w).Header().Set("Access-Control-Expose-Headers", "Authorization")
	(*w).Header().Set("Access-Control-Allow-Credentials", "true")
}

//Jwt handler to issue the token and set cookie, from a couple of username / password
func CredsHandler(w http.ResponseWriter, r *http.Request) {
	//handle CORS: add headers and check options (return 204 to let the other request be done)
	addCors(&w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

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
		addCors(&w)
		var u User

		if isEventStream(r) { //Cookie part (for EventStream API auth)
			c, err := r.Cookie("sse_token")
			if err != nil {
				if err == http.ErrNoCookie { //no cookie, no token
					w.WriteHeader(http.StatusUnauthorized)
					io.WriteString(w, `{"error":"No cookie with token"}`)
					return
				}
				w.WriteHeader(http.StatusBadRequest)
				io.WriteString(w, `{"error":"Bad request"}`)
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
			json.Unmarshal([]byte(cookieValue), &u)

		} else { //Token is in authorisation header's bearer string
			token := getTokenFrom(r)
			if token == "" {
				w.WriteHeader(http.StatusUnauthorized)
				io.WriteString(w, `{"error":"Token not found"}`)
				return
			}

			//hydrate user with token datas
			u.Token = token
		}

		//check token's validity
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(u.Token, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})
		if err != nil {
			if err == jwt.ErrSignatureInvalid {
				w.WriteHeader(http.StatusUnauthorized)
				io.WriteString(w, `{"error":"Invalid token signature"}`)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, `{"error":"Bad request or expired token"}`)
			return
		}
		if !token.Valid {
			w.WriteHeader(http.StatusUnauthorized)
			io.WriteString(w, `{"error":"Invalid token"}`)
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
	//ensure creds are sets
	if username == "" || password == "" {
		return false
	}

	log.Println(os.Getenv("SSE_BROADCASTER_USERNAME"))
	log.Println(os.Getenv("SSE_BROADCASTER_PASSWORD"))
	isClientOk := (username == os.Getenv("SSE_CLIENT_USERNAME") && password == os.Getenv("SSE_CLIENT_PASSWORD"))
	isBroadcasterOk := (username == os.Getenv("SSE_BROADCASTER_USERNAME") && password == os.Getenv("SSE_BROADCASTER_PASSWORD"))

	return isClientOk || isBroadcasterOk
}

//expiresAt in one hour ?
func expiresAt() time.Time {
	return time.Now().Add(time.Hour * time.Duration(1))
}

//isEventStream returns true/false if event stream header was or was not found in request
func isEventStream(r *http.Request) bool {
	h := r.Header.Get("Accept")
	return h != "" && h == "text/event-stream"
}

//getTokenFrom request, parsing the auhtorization bearer and returning it if found
func getTokenFrom(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h != "" {
		var re = regexp.MustCompile(`(?mi)^Bearer\s+(.*)`)
		matches := re.FindStringSubmatch(h)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}
