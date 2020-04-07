package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sse/auth"
	"sse/broker"
	"sse/example"

	"github.com/joho/godotenv"
)

var b broker.Broker       //the one that holds the channels and know where to dispatch events
var bc broker.Broadcaster //the one that knows how to broadcast events form the outside to the inside

func main() {
	//routing
	http.Handle("/login", http.HandlerFunc(auth.CredsHandler))        //issue token from username/password
	http.Handle("/broadcast", auth.AuthMiddleware(bc.WithBroker(&b))) //wrapper to pass extra args
	http.Handle("/events", auth.AuthMiddleware(&b))                   //will call auth middleware, then ServeHTTP method (default)
	http.Handle("/example", http.HandlerFunc(example.MainHandler))    //classic handler for example

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

//instantiate a Broker and a Broadcaster
func init() {
	godotenv.Load()

	if os.Getenv("SSE_CLIENT_USERNAME") == "" || os.Getenv("SSE_CLIENT_PASSWORD") == "" {
		log.Fatal("Missing user sse's user creds")
	}

	if os.Getenv("SSE_BROADCASTER_USERNAME") == "" || os.Getenv("SSE_BROADCASTER_PASSWORD") == "" {
		log.Fatal("Missing sse's broadcaster creds")
	}

	b = broker.Broker{
		Clients:     make(map[chan broker.Message]broker.Client),
		NewClients:  make(chan broker.Client),
		DcnxClients: make(chan broker.Client),
		Messages:    make(chan broker.Message),
	}

	b.Start()

	bc = broker.Broadcaster{}
}
