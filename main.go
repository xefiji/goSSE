package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sse/auth"
	"sse/broker"
	"sse/example"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

var b broker.Broker       //the one that holds the channels and know where to dispatch events
var bc broker.Broadcaster //the one that knows how to broadcast events form the outside to the inside

func main() {
	//routing
	http.Handle("/login", http.HandlerFunc(auth.CredsHandler))        //issue token from username/password
	http.Handle("/broadcast", auth.AuthMiddleware(bc.WithBroker(&b))) //wrapper to pass extra args
	http.Handle("/events", auth.AuthMiddleware(&b))                   //will call auth middleware, then ServeHTTP method (default)
	http.Handle("/admin/stop", auth.AuthMiddleware(http.HandlerFunc(b.Stop)))
	http.Handle("/example", http.HandlerFunc(example.MainHandler)) //classic handler for example

	//serving. todo: port as var
	portVar := os.Getenv("SSE_PORT")
	if portVar == "" {
		portVar = "80"
	}

	log.Printf("Running on port %s\n\n", portVar)

	srv := &http.Server{
		Addr: fmt.Sprintf(":%s", portVar),
	}

	go func() {
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatal("❌ failed to start server with error: ", err.Error())
		}
	}()

	signal.Notify(auth.Quit, syscall.SIGINT, syscall.SIGTERM)
	<-auth.Quit

	log.Println("Shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := srv.Shutdown(ctx)
	if err != nil {
		log.Fatal("❌ failed to shut down server with error: ", err.Error())
	}

	log.Println("✅ Server shut down gracefully")
	os.Exit(0)
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
