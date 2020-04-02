package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sse/auth"
	"sse/broker"
	"text/template"
)

var b broker.Broker
var bc broker.Broadcaster
var basepath string

func main() {
	//routing
	http.Handle("/login", http.HandlerFunc(auth.TokenHandler))
	http.Handle("/broadcast", bc.WithBroker(&b))    //wrapper to pass extra args
	http.Handle("/events", auth.AuthMiddleware(&b)) //will call ServeHTTP method (default)
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
