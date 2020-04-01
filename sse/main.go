package main

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"runtime"
	"sse/broker"
	"text/template"
)

var b broker.Broker
var bc broker.Broadcaster
var basepath string

func main() {
	//routing
	http.Handle("/broadcast", bc.WithBroker(&b))    //wrapper to pass extra args
	http.Handle("/events", &b)                      //will call ServeHTTP method (default)
	http.Handle("/", http.HandlerFunc(mainHandler)) //classic handler

	//serving
	http.ListenAndServe(":80", nil)
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
