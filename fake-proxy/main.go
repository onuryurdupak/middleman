package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	fmt.Println("Starting proxy.")
	handler := &httpHandler{}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nStopping proxy.")
		os.Exit(1)
	}()

	http.ListenAndServe(":8080", handler)

}

type httpHandler struct{}

func (h *httpHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	// TODO:
	// Read url, headers and payload.
	// Get reroute url suffix and make identical request.
	// Build identical response from routed source's response.
	// Return response to client.
}
