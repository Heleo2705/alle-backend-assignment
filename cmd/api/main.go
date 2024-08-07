package main

import (
	"context"
	"errors"
	"log"

	"fmt"
	"net/http"
	"time"

	"github.com/r3labs/sse/v2"
)

func main() {
	sseServer := sse.New()
	sseServer.AutoReplay = false
	sseServer.Headers = map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
	}
	mux := http.NewServeMux()
	sseServer.CreateStream("messages")

	sseServer.Publish("messages", &sse.Event{
		Data: []byte(time.Now().String()),
	})
	mux.HandleFunc("/messages", func(w http.ResponseWriter, r *http.Request) {

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second) // if the inference provider takes more than 5 seconds, we can check and respond accordingly
		defer cancel()
		go func() {
			for {
				select {

				case <-time.After(1 * time.Second): // mock inference provider
					sseServer.Publish("messages", &sse.Event{
						Data: []byte(time.Now().String()),
					})

				case <-ctx.Done(): // means that either the client disconnected or the inference provider failer. Here we can give recovery options after caching last sent results
					fmt.Println("request was timed out or cancelled. Do something after this")
					err := ctx.Err()
					if err != nil {
						fmt.Println(ctx.Err())
						if errors.Is(ctx.Err(), context.DeadlineExceeded) {
							fmt.Println("switching inference provider")
							// switch your inference provider here if there has been no response in 5 seconds and restart the process
						} else if errors.Is(ctx.Err(), context.Canceled) {
							fmt.Println("caching response")
							//cache response in database for future resumability
						}
					}
					return
				}
			}
		}()
		sseServer.ServeHTTP(w, r.WithContext(ctx))
	})
	fmt.Printf("Running on 8080")
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatal(err)
	}
}
