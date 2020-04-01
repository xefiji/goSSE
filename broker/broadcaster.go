package broker

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Message struct {
	Content string
}

type Broadcaster struct {
	Name string
}

// actual broadcast to broker messages channel
func (broadcaster *Broadcaster) broadcast(msg Message, b *Broker) {
	b.Messages <- fmt.Sprintf("%s", msg.Content)
}

// for tests
func (broadcaster *Broadcaster) broadcastLoop(b *Broker) {
	fmt.Println("Starting to broadcast in a loop")
	for i := 0; ; i++ {
		log.Printf("Sent message %d ", i)
		b.Messages <- fmt.Sprintf("%d - time is %v", i, time.Now())
		time.Sleep(5e9)
	}
}

//wrap handler to pass broker
func (broadcaster *Broadcaster) WithBroker(b *Broker) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m Message
		err := json.NewDecoder(r.Body).Decode(&m)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		broadcaster.broadcast(m, b)
		w.WriteHeader(http.StatusCreated)
	})
}
