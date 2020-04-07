package broker

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

type Message struct {
	Content string
	User    string
	Type    string
}

type Broadcaster struct {
	Name string
}

// actual broadcast to broker messages channel
func (broadcaster *Broadcaster) broadcast(msg Message, b *Broker) {
	b.Messages <- msg
}

// for tests
func (broadcaster *Broadcaster) broadcastLoop(b *Broker) {
	log.Println("Starting to broadcast in a loop")
	for i := 0; ; i++ {
		log.Printf("Sent message %d ", i)
		m := Message{fmt.Sprintf("%d - time is %v", i, time.Now()), strconv.Itoa(rand.Intn(100)), "sse"}
		b.Messages <- m
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
