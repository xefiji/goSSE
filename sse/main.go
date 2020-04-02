package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sse/broker"
	"text/template"
	"time"

	"github.com/dgrijalva/jwt-go"
)

var b broker.Broker
var bc broker.Broadcaster
var basepath string
var jwtKey = []byte(os.Getenv("SSE_APP_KEY"))

type User struct {
	Name     string `json:"username"`
	Password string `json:"password"`
	Token    string
}

type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

func main() {
	//routing
	http.Handle("/login", http.HandlerFunc(TokenHandler))
	http.Handle("/broadcast", bc.WithBroker(&b))    //wrapper to pass extra args
	http.Handle("/events", AuthMiddleware(&b))      //will call ServeHTTP method (default)
	http.Handle("/", http.HandlerFunc(mainHandler)) //classic handler

	fmt.Println("Running...")

	//serving. todo: port as var
	if err := http.ListenAndServe(":80", nil); err != nil {
		log.Fatal(err)
	}
}

//Serves templates with SSE js object (mainly for tests)
func mainHandler(w http.ResponseWriter, r *http.Request) {
	//force path to "/" only
	if r.URL.Path != "/" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	//Read template
	t, err := template.ParseFiles(fmt.Sprintf("%s\\templates\\index.html", basepath))
	if err != nil {
		log.Fatalf("Error parsing template: %s", err)
	}

	//Render template
	var env_vars = map[string]string{"user": os.Getenv("SSE_USERNAME"), "password": os.Getenv("SSE_PASSWORD")}
	t.Execute(w, env_vars)
}

//instantiate a Broker and a Broadcaster
func init() {
	b = broker.Broker{
		Clients:     make(map[chan string]bool),
		NewClients:  make(chan (chan string)),
		DcnxClients: make(chan (chan string)),
		Messages:    make(chan string),
	}

	b.Start()

	bc = broker.Broadcaster{}

	_, t, _, _ := runtime.Caller(0)
	basepath = filepath.Dir(t)
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
