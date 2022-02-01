package main

import (
	"context"
	"log"
	"net/http"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}
var rd = redis.NewClient(&redis.Options{
	Addr: "localhost:6379",
})
var ctx = context.Background()

// should handle more errors
// deadlock condition?
func echo(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("websocket connection err:", err)
		return
	}
	defer conn.Close()

	room := make(chan string)
	start := make(chan string)

	go func() {
	loop:
		for {
			sub := rd.Subscribe(ctx)
			// subCh := sub.Channel()
			defer sub.Close()

			for {
				select {
				case channel := <-room:
					log.Println("channel", channel)
					sub = rd.Subscribe(ctx, channel)
					// _, err := sub.Receive(ctx)
					// if err != nil {
					// 	log.Println("redis sub connection err:", err)
					// 	break loop
					// }
					start <- "ok"
				case msg := <-sub.Channel():
					log.Println("msg", msg)
					err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload))
					if err != nil {
						log.Println("websocket write err:", err)
						break loop
					}
				}
			}

		}
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("websocket read err:", err)
			break
		}
		log.Println(string(msg))

		ch := "test-channel"
		if string(msg) == "test" {
			room <- ch
			log.Println(ch)
			log.Println(<-start)
		}

		if err := rd.Publish(ctx, ch, msg).Err(); err != nil {
			log.Println("redis publish err:", err)
			break
		}
	}

}

func main() {
	http.Handle("/", http.FileServer(http.Dir("./js")))
	http.HandleFunc("/ws", echo)

	log.Println("server starting...", "http://localhost:5000")
	log.Fatal(http.ListenAndServe("localhost:5000", nil))
}
