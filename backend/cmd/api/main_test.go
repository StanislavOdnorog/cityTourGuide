package main

import (
	"bytes"
	"context"
	"io"
	"errors"
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/handler"
)

type testPinger struct{}

func (testPinger) Ping(context.Context) error {
	return nil
}

func TestServeWithGracefulShutdown_DrainsInflightRequestsAndStopsNewConnections(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	healthHandler := handler.NewHealthHandler(testPinger{})
	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	cleanupCalled := make(chan struct{})

	router := gin.New()
	router.GET("/readyz", healthHandler.Readyz)
	router.GET("/slow", func(c *gin.Context) {
		close(requestStarted)
		<-releaseRequest
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	srv := &http.Server{Handler: router}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	sigCh := make(chan os.Signal, 2)
	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- serveWithGracefulShutdown(
			srv,
			listener,
			sigCh,
			healthHandler.SetShuttingDown,
			5*time.Second,
			func(code int) {
				t.Errorf("unexpected forceExit(%d)", code)
			},
			func() error {
				close(cleanupCalled)
				return nil
			},
		)
	}()

	baseURL := "http://" + listener.Addr().String()
	client := &http.Client{}

	respCh := make(chan *http.Response, 1)
	reqErrCh := make(chan error, 1)
	go func() {
		resp, err := client.Get(baseURL + "/slow")
		if err != nil {
			reqErrCh <- err
			return
		}
		respCh <- resp
	}()

	select {
	case <-requestStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for slow request to start")
	}

	sigCh <- syscall.SIGTERM

	select {
	case <-cleanupCalled:
		t.Fatal("cleanup ran before in-flight request completed")
	default:
	}

	shutdownClient := &http.Client{Transport: &http.Transport{DisableKeepAlives: true}}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := shutdownClient.Get(baseURL + "/readyz")
		if err != nil {
			goto listenerClosed
		}
		resp.Body.Close()
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("server kept accepting new connections after shutdown started")

listenerClosed:

	close(releaseRequest)

	select {
	case err := <-reqErrCh:
		t.Fatalf("slow request failed: %v", err)
	case resp := <-respCh:
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("slow request status = %d, want 200", resp.StatusCode)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for slow request to complete")
	}

	select {
	case err := <-serveErrCh:
		if err != nil {
			t.Fatalf("serveWithGracefulShutdown() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for server shutdown")
	}

	select {
	case <-cleanupCalled:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("cleanup was not called after shutdown")
	}

	conn, err := net.DialTimeout("tcp", listener.Addr().String(), 200*time.Millisecond)
	if err == nil {
		conn.Close()
		t.Fatal("expected listener to reject new connections after shutdown")
	}
}

func TestServeWithGracefulShutdown_SecondSignalForcesExit(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	var shuttingDown atomic.Bool
	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	forceExitCh := make(chan int, 1)

	router := gin.New()
	router.GET("/slow", func(c *gin.Context) {
		close(requestStarted)
		<-releaseRequest
		c.Status(http.StatusOK)
	})

	srv := &http.Server{Handler: router}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	sigCh := make(chan os.Signal, 2)
	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- serveWithGracefulShutdown(
			srv,
			listener,
			sigCh,
			shuttingDown.Store,
			5*time.Second,
			func(code int) {
				forceExitCh <- code
			},
		)
	}()

	baseURL := "http://" + listener.Addr().String()
	go func() {
		resp, err := http.Get(baseURL + "/slow")
		if err == nil {
			resp.Body.Close()
		}
	}()

	select {
	case <-requestStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for slow request to start")
	}

	sigCh <- syscall.SIGTERM
	sigCh <- syscall.SIGTERM

	select {
	case code := <-forceExitCh:
		if code != 1 {
			t.Fatalf("forceExit code = %d, want 1", code)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("second signal did not trigger force exit")
	}

	close(releaseRequest)

	select {
	case err := <-serveErrCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("serveWithGracefulShutdown() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for server shutdown")
	}
}

func TestLimitRequestBodySize_RejectsBodiesOver1MB(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(limitRequestBodySize(maxRequestBodySize))
	router.POST("/echo", func(c *gin.Context) {
		_, err := io.Copy(io.Discard, c.Request.Body)
		if err != nil {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		c.Status(http.StatusNoContent)
	})

	srv := &http.Server{Handler: router}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- srv.Serve(listener)
	}()
	defer func() {
		_ = srv.Close()
		if err := <-serveErrCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("Serve() error = %v", err)
		}
	}()

	req, err := http.NewRequest(http.MethodPost, "http://"+listener.Addr().String()+"/echo", bytes.NewReader(bytes.Repeat([]byte("a"), maxRequestBodySize+1)))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusRequestEntityTooLarge)
	}
}
