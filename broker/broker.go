package broker

import (
	"fmt"
	"log"
	"net/http"
	"sse/auth"
	"syscall"
)

const (
	allEventType = "all"
)

//holds sse headers
var headers map[string]string

//holds Client (connected instance) infos and message chan
type Client struct {
	Id          string
	Type        string
	MessageChan chan Message
}

//this struct will hold all channels
type Broker struct {
	Clients     map[chan Message]Client //map of client's channels
	NewClients  chan Client             //each new client channel is pushed as a channel in here
	DcnxClients chan Client             //each disconnected client is pushed here to be removed from Clients
	Messages    chan Message            //each sent message (event) is pushed here
}

//Sets common SSE headers
func init() {
	headers = make(map[string]string)
	headers["Content-Type"] = "text/event-stream"
	headers["Cache-Control"] = "no-cache"
	headers["Connection"] = "keep-alive"
	headers["Transfer-Encoding"] = "chunked"
}

//Start starts a new goroutine that handles new clients, client's dcnx and SSE broadcast
func (b *Broker) Start() {
	go func() {
		for {
			//Block until we receive from one of the three following channels.
			select {
			case s := <-b.NewClients: //add new client
				b.Clients[s.MessageChan] = s
				log.Printf("Client connected. Total: %d\n", len(b.Clients))

			case s := <-b.DcnxClients: //remove disconnected client and close its channel
				delete(b.Clients, s.MessageChan)
				close(s.MessageChan)
				log.Printf("Client disconnected. Total: %d\n", len(b.Clients))

			case msg := <-b.Messages: //send new message to each attached client
				go b.dispatch(msg)
			}
		}
	}()
}

//Stop send a sigterm to Quit signal to shutdown server gracefully
func (b *Broker) Stop(w http.ResponseWriter, r *http.Request) {
	log.Println("Stopping route called...")
	auth.Quit <- syscall.SIGTERM
}

//dispatch sends message to all targeted clients
func (b *Broker) dispatch(msg Message) {
	total := 0
	for _, c := range b.Clients {
		if msg.User != "" { //send only to this user
			if msg.User == c.Id {
				c.MessageChan <- msg
				total++
			}
		} else if msg.Type != "" { //send only to clients attached to this type channel
			if msg.Type == c.Type {
				c.MessageChan <- msg
				total++
			}
		} else { //send to all
			c.MessageChan <- msg
			total++
		}
	}

	log.Printf("Broadcasted %s message to %d clients\n", msg.Type, total)
}

//handler
func (b *Broker) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	//check if streaming is supported
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	//type of event for which the user wants to receive messages
	eventType := r.URL.Query().Get("type")
	if eventType == "" {
		eventType = allEventType
	}

	//always need a user
	user := r.Context().Value("user")
	if user == nil {
		http.Error(w, "No user found!", http.StatusForbidden)
		return
	}

	//user is an interface, type assert it
	//see: https: //golang.org/doc/effective_go.html#interface_conversions
	var userId string
	userId = user.(auth.User).Id

	//SSE client has just arrived: create a new channel and attach it to the client's map
	messageChan := make(chan Message)
	newClient := Client{userId, eventType, messageChan} //@todo put ID Here
	b.NewClients <- newClient

	//check if client has disconnected
	go func() {
		<-r.Context().Done()
		b.DcnxClients <- newClient //client will be removed hence not receiving anything
	}()

	//set sse headers
	for k, v := range headers {
		w.Header().Set(k, v)
	}

	for { //persistent connexion

		msg, open := <-messageChan //read from client's channel
		if !open {
			break //a closed channel means a disconnected client
		}

		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", msg.Type, msg.Content) //write datas
		f.Flush()                                                        //flush response
	}

	log.Println("Finished HTTP request serving SSEs at ", r.URL.Path)
}
