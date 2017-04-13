package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"net/http"

	"golang.org/x/net/context"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := &Config{
		Interval: time.Second * 60,
		Region:   "us-east-1",
		Service:  "ec2",
		Confirm: func(w http.ResponseWriter, r *http.Request) bool {
			// whitelist / log requests
			return true
		},
	}

	blocker := New(config)
	blocker.Start(ctx, http.DefaultClient)

	m := http.NewServeMux()

	m.HandleFunc("/", index)

	h := blocker.Middleware(m)

	log.Fatal(http.ListenAndServe(":8080", h))
}

func index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "index page")
}
