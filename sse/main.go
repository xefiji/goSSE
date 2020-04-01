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

type User struct {
	Name     string `json:"username"`
	Password string `json:"password"`
	Token    string
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
	t.Execute(w, nil)
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
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var u User
	err := json.NewDecoder(r.Body).Decode(&u)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//todo make it stronger
	if u.Name != os.Getenv("SSE_USERNAME") || u.Password != os.Getenv("SSE_PASSWORD") {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"error":"Invalid creds"}`)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": u.Name,
		"exp":  time.Now().Add(time.Hour * time.Duration(1)).Unix(), //one hour
		"iat":  time.Now().Unix(),
	})

	tokenString, err := token.SignedString([]byte(os.Getenv("SSE_APP_KEY")))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, `{"error":"Token generation failed"}`)
		return
	}

	tokenString = fmt.Sprintf("{\"token\":\"%s\"}", tokenString)
	io.WriteString(w, tokenString)
	return
}

//handler wrapper for jwt auth
func AuthMiddleware(next http.Handler) http.Handler {
	if len(os.Getenv("SSE_APP_KEY")) == 0 {
		log.Fatal("Missing APP_KEY")
	}

	//todo do the middleware check for auth here
	return next //next should be the broker, in our case
}
