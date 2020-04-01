package broker

import (
	"fmt"
	"log"
	"net/http"
)

var headers map[string]string

//this struct will hold all channels
type Broker struct {
	Clients     map[chan string]bool //map of client's channels
	NewClients  chan chan string     //each new client channel is pushed as a channel in here
	DcnxClients chan chan string     //each disconnected client is pushed here to be removed from Clients
	Messages    chan string          //each sent message (event) is pushed here
}

//Sets common SSE headers
func init() {
	headers = make(map[string]string)
	headers["Content-Type"] = "text/event-stream"
	headers["Cache-Control"] = "no-cache"
	headers["Connection"] = "keep-alive"
	headers["Transfer-Encoding"] = "chunked"
}

//starts a new goroutine that handles new clients, client's dcnx and SSE broadcast
func (b *Broker) Start() {
	go func() {
		for {
			//Block until we receive from one of the three following channels.
			select {

			case s := <-b.NewClients: //add new client
				b.Clients[s] = true
				log.Println("Client connected")

			case s := <-b.DcnxClients: //remove disconnected client and close its channel
				delete(b.Clients, s)
				close(s)
				log.Println("Client disconnected")

			case msg := <-b.Messages: //send new message to each attached client
				for s := range b.Clients {
					s <- msg
				}
				log.Printf("Broadcasted to %d clients", len(b.Clients))
			}
		}
	}()
}

//handler
func (b *Broker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//check if streaming is supported
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	//SSE client has just arrived: create a new channel and attach it to the client's map
	messageChan := make(chan string)
	b.NewClients <- messageChan

	//check if client has disconnected
	go func() {
		<-r.Context().Done()
		b.DcnxClients <- messageChan //client will be removed hence not receiving anything
	}()

	for k, v := range headers { //set sse headers
		w.Header().Set(k, v)
	}

	for { //persistent connexion

		msg, open := <-messageChan //read from client's channel
		if !open {
			break //a closed channel means a disconnected client
		}

		fmt.Fprintf(w, "data: %s\n\n", msg) //write datas
		f.Flush()                           //flush response
	}

	log.Println("Finished HTTP request serving SSEs at ", r.URL.Path)
}
