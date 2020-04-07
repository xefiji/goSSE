package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sse/auth"
	"sse/broker"

	"github.com/joho/godotenv"
)

var b broker.Broker       //the one that holds the channels and know where to dispatch events
var bc broker.Broadcaster //the one that knows how to broadcast events form the outside to the inside
var tplpath string        //basepath for template parsing.

func main() {
	//routing
	http.Handle("/login", http.HandlerFunc(auth.TokenHandler))
	http.Handle("/broadcast", auth.AuthMiddleware(bc.WithBroker(&b))) //wrapper to pass extra args
	http.Handle("/events", auth.AuthMiddleware(&b))                   //will call auth middleware, then ServeHTTP method (default)
	http.Handle("/", http.HandlerFunc(mainHandler))                   //classic handler

	log.Println("Running...")

	//serving. todo: port as var
	portVar := os.Getenv("SSE_PORT")
	if portVar == "" {
		portVar = "80"
	}
	if err := http.ListenAndServe(fmt.Sprintf(":%s", portVar), nil); err != nil {
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
	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Panicf("Error parsing template: %s", err)
	}

	//Render template
	var env_vars = map[string]string{"user": os.Getenv("SSE_USERNAME"), "password": os.Getenv("SSE_PASSWORD")}
	t.Execute(w, env_vars)
}

//instantiate a Broker and a Broadcaster
//sets tplpath for template parsing. todo: should be done in a better
func init() {
	godotenv.Load()

	b = broker.Broker{
		Clients:     make(map[chan broker.Message]broker.Client),
		NewClients:  make(chan broker.Client),
		DcnxClients: make(chan broker.Client),
		Messages:    make(chan broker.Message),
	}

	b.Start()

	bc = broker.Broadcaster{}
}
