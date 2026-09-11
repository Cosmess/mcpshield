package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestShutdownServerHonorsDeadline(t *testing.T) {
	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	server := &http.Server{Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		close(requestStarted)
		<-releaseRequest
	})}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()

	clientDone := make(chan error, 1)
	go func() {
		_, requestErr := http.Get("http://" + listener.Addr().String())
		clientDone <- requestErr
	}()
	select {
	case <-requestStarted:
	case <-time.After(time.Second):
		t.Fatal("request did not reach handler")
	}

	err = shutdownServer(server, 20*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("shutdownServer() error = %v, want deadline exceeded", err)
	}
	close(releaseRequest)
	_ = server.Close()
	<-serveDone
}
