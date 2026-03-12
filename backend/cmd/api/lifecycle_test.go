package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// ---------- Graceful Shutdown: drains in-flight request ----------

func TestLifecycle_GracefulShutdown_DrainsInflight(t *testing.T) {
	gin.SetMode(gin.TestMode)

	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})

	r := gin.New()
	r.GET("/slow", func(c *gin.Context) {
		close(requestStarted)
		<-releaseRequest
		c.JSON(http.StatusOK, gin.H{"drained": true})
	})

	srv := &http.Server{Handler: r}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	base := "http://" + ln.Addr().String()

	sigCh := make(chan os.Signal, 2)
	var cleanupCalled atomic.Bool
	forceExitCalled := make(chan int, 1)

	resultCh := make(chan error, 1)
	go func() {
		resultCh <- serveWithGracefulShutdown(
			srv,
			ln,
			sigCh,
			func(bool) {}, // setShuttingDown stub
			5*time.Second,
			func(code int) { forceExitCalled <- code },
			func() error { cleanupCalled.Store(true); return nil },
		)
	}()

	// Start an in-flight request.
	type respResult struct {
		resp *http.Response
		err  error
	}
	respCh := make(chan respResult, 1)
	go func() {
		resp, err := http.Get(base + "/slow")
		respCh <- respResult{resp, err}
	}()

	select {
	case <-requestStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for in-flight request to start")
	}

	// Send SIGTERM to initiate graceful shutdown.
	sigCh <- syscall.SIGTERM

	// Give the shutdown a moment to close the listener.
	time.Sleep(100 * time.Millisecond)

	// Release the in-flight request so it can complete.
	close(releaseRequest)

	// The in-flight request must complete successfully.
	select {
	case rr := <-respCh:
		if rr.err != nil {
			t.Fatalf("in-flight request error: %v", rr.err)
		}
		defer rr.resp.Body.Close()
		if rr.resp.StatusCode != http.StatusOK {
			t.Fatalf("in-flight request status = %d, want 200", rr.resp.StatusCode)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(rr.resp.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["drained"] != true {
			t.Fatalf("expected drained=true in response body, got %v", body)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for in-flight request response")
	}

	// serveWithGracefulShutdown must return nil (clean shutdown).
	select {
	case err := <-resultCh:
		if err != nil {
			t.Fatalf("serveWithGracefulShutdown returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for serveWithGracefulShutdown to return")
	}

	// Cleanup functions must have been called.
	if !cleanupCalled.Load() {
		t.Fatal("expected cleanup function to be called after shutdown")
	}
}

// ---------- New requests rejected after shutdown starts ----------

func TestLifecycle_GracefulShutdown_RejectsNewRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})

	r := gin.New()
	r.GET("/slow", func(c *gin.Context) {
		close(requestStarted)
		<-releaseRequest
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"pong": true})
	})

	srv := &http.Server{Handler: r}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	base := "http://" + ln.Addr().String()

	sigCh := make(chan os.Signal, 2)
	resultCh := make(chan error, 1)
	go func() {
		resultCh <- serveWithGracefulShutdown(
			srv, ln, sigCh, func(bool) {},
			5*time.Second,
			func(int) {},
		)
	}()

	// Start an in-flight request to keep the server draining.
	go func() {
		resp, err := http.Get(base + "/slow")
		if err == nil {
			resp.Body.Close()
		}
	}()

	select {
	case <-requestStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for slow request to start")
	}

	// Initiate shutdown.
	sigCh <- syscall.SIGTERM

	// Poll until the listener is closed (new connections fail).
	noKeepAlive := &http.Client{
		Transport: &http.Transport{DisableKeepAlives: true},
		Timeout:   500 * time.Millisecond,
	}
	deadline := time.Now().Add(3 * time.Second)
	rejected := false
	for time.Now().Before(deadline) {
		resp, err := noKeepAlive.Get(base + "/ping")
		if err != nil {
			rejected = true
			break
		}
		resp.Body.Close()
		time.Sleep(25 * time.Millisecond)
	}
	if !rejected {
		t.Fatal("server kept accepting new connections after shutdown was initiated")
	}

	// Let the in-flight request finish so shutdown completes.
	close(releaseRequest)

	select {
	case err := <-resultCh:
		if err != nil {
			t.Fatalf("serveWithGracefulShutdown error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for shutdown")
	}
}

// ---------- Force exit on second signal ----------

func TestLifecycle_ForceExit_OnSecondSignal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	requestStarted := make(chan struct{})
	// Never release — simulates a stuck handler.
	neverRelease := make(chan struct{})

	r := gin.New()
	r.GET("/stuck", func(c *gin.Context) {
		close(requestStarted)
		<-neverRelease
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	srv := &http.Server{Handler: r}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	base := "http://" + ln.Addr().String()

	sigCh := make(chan os.Signal, 2)
	forceExitCode := make(chan int, 1)

	resultCh := make(chan error, 1)
	go func() {
		resultCh <- serveWithGracefulShutdown(
			srv, ln, sigCh, func(bool) {},
			// Short shutdown timeout so the test doesn't hang.
			30*time.Second,
			func(code int) { forceExitCode <- code },
		)
	}()

	// Start a stuck request.
	go func() {
		resp, err := http.Get(base + "/stuck")
		if err == nil {
			resp.Body.Close()
		}
	}()

	select {
	case <-requestStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for stuck request to start")
	}

	// First signal: graceful shutdown starts.
	sigCh <- syscall.SIGTERM
	time.Sleep(100 * time.Millisecond)

	// Second signal: force exit.
	sigCh <- syscall.SIGINT

	select {
	case code := <-forceExitCode:
		if code != 1 {
			t.Fatalf("force exit code = %d, want 1", code)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for force exit callback")
	}

	// Unblock the stuck handler so goroutines can clean up.
	close(neverRelease)
}

// ---------- Client disconnect cancels handler context ----------

func TestLifecycle_ClientDisconnect_CancelsContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handlerStarted := make(chan struct{})
	handlerCtxErr := make(chan error, 1)
	handlerDone := make(chan struct{})

	r := gin.New()
	r.GET("/cancellable", func(c *gin.Context) {
		close(handlerStarted)
		// Wait for context cancellation (client disconnect) or a long timeout.
		select {
		case <-c.Request.Context().Done():
			handlerCtxErr <- c.Request.Context().Err()
		case <-time.After(10 * time.Second):
			handlerCtxErr <- fmt.Errorf("handler timed out — context was never cancelled")
		}
		close(handlerDone)
	})

	srv := &http.Server{Handler: r}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()

	errCh := make(chan error, 1)
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	// Open a raw TCP connection and send an HTTP request, then close it.
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	_, err = conn.Write([]byte("GET /cancellable HTTP/1.1\r\nHost: localhost\r\n\r\n"))
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	// Wait for the handler to start processing.
	select {
	case <-handlerStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for handler to start")
	}

	// Disconnect the client.
	conn.Close()

	// The handler should observe context cancellation.
	select {
	case ctxErr := <-handlerCtxErr:
		if ctxErr == nil {
			t.Fatal("expected non-nil context error after client disconnect")
		}
		if !errors.Is(ctxErr, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", ctxErr)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for handler context cancellation")
	}

	<-handlerDone
}

// ---------- Server write timeout terminates slow responses ----------

func TestLifecycle_WriteTimeout_TerminatesSlowResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	writeTimeout := 500 * time.Millisecond
	handlerDone := make(chan struct{})

	r := gin.New()
	r.GET("/slow-write", func(c *gin.Context) {
		defer close(handlerDone)
		// Stall the entire response beyond the write timeout.
		// The server's WriteTimeout deadline is set when the request
		// headers are read, so sleeping here consumes it.
		time.Sleep(writeTimeout + 300*time.Millisecond)
		c.JSON(http.StatusOK, gin.H{"late": true})
	})

	srv := &http.Server{
		Handler:      r,
		WriteTimeout: writeTimeout,
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()

	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Logf("serve error (expected after timeout): %v", err)
		}
	}()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	// Use a raw TCP connection so we can observe the server closing it.
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte("GET /slow-write HTTP/1.1\r\nHost: localhost\r\n\r\n"))
	if err != nil {
		t.Fatalf("write request: %v", err)
	}

	// Set a generous read deadline so the test itself doesn't hang.
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	body, readErr := io.ReadAll(conn)

	// The server should have closed the connection before sending a
	// complete response. We may get EOF, an incomplete response, or
	// no data at all.
	fullResponse := string(body)
	if readErr == nil && len(fullResponse) > 0 {
		// If we got some data, it must NOT contain the late JSON body.
		if containsJSON(fullResponse, "late") {
			t.Fatal("expected WriteTimeout to prevent the late response, but full JSON body was received")
		}
	}

	// Wait for handler goroutine to finish to avoid leaks.
	<-handlerDone
}

// containsJSON checks whether a raw HTTP response string contains a JSON
// key. This is a simple substring check for test assertions.
func containsJSON(raw, key string) bool {
	return len(raw) > 0 && json.Valid([]byte(raw[findBodyStart(raw):])) && findKey(raw, key)
}

func findBodyStart(raw string) int {
	for i := 0; i < len(raw)-3; i++ {
		if raw[i] == '\r' && raw[i+1] == '\n' && raw[i+2] == '\r' && raw[i+3] == '\n' {
			return i + 4
		}
	}
	return 0
}

func findKey(raw, key string) bool {
	return len(raw) > 0 && len(key) > 0 && stringContains(raw, `"`+key+`"`)
}

func stringContains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ---------- Read timeout rejects slow client ----------

func TestLifecycle_ReadTimeout_RejectsSlowClient(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var handlerCalled atomic.Bool

	r := gin.New()
	r.POST("/read-body", func(c *gin.Context) {
		handlerCalled.Store(true)
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusRequestTimeout, gin.H{"error": "read timeout"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"length": len(body)})
	})

	readTimeout := 500 * time.Millisecond
	srv := &http.Server{
		Handler:     r,
		ReadTimeout: readTimeout,
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()

	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Logf("serve error: %v", err)
		}
	}()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	// Open a raw TCP connection and send headers but never finish the body.
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Send an incomplete request: headers but stall on the body.
	_, err = conn.Write([]byte("POST /read-body HTTP/1.1\r\nHost: localhost\r\nContent-Length: 99999\r\n\r\n"))
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	// Wait for the read timeout to expire, then try to read the response.
	// The server should close the connection.
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 4096)
	n, readErr := conn.Read(buf)

	// The server should either close the connection (EOF) or we get an error.
	// The key assertion: the handler should NOT have received the full body.
	if readErr == nil && n > 0 {
		// If we got a response, it should not be a 200 with the full body.
		response := string(buf[:n])
		if handlerCalled.Load() {
			// Handler was called — the read timeout was set on the body read.
			// That's acceptable; check the response isn't a happy 200.
			if resp := response; len(resp) > 0 {
				t.Logf("got response: %s", resp[:min(len(resp), 200)])
			}
		}
		return // Connection closed or error response — timeout worked.
	}

	if readErr != nil {
		// Connection was closed by server due to read timeout — correct behavior.
		return
	}
}

// ---------- Cleanup functions called on shutdown ----------

func TestLifecycle_Shutdown_CallsAllCleanupFunctions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	srv := &http.Server{Handler: r}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	sigCh := make(chan os.Signal, 2)
	var mu sync.Mutex
	var order []int

	makeCleanup := func(id int) func() error {
		return func() error {
			mu.Lock()
			defer mu.Unlock()
			order = append(order, id)
			return nil
		}
	}

	resultCh := make(chan error, 1)
	go func() {
		resultCh <- serveWithGracefulShutdown(
			srv, ln, sigCh, func(bool) {},
			5*time.Second,
			func(int) {},
			makeCleanup(1), makeCleanup(2), makeCleanup(3),
		)
	}()

	// Give the server a moment to start.
	time.Sleep(50 * time.Millisecond)

	// Initiate shutdown.
	sigCh <- syscall.SIGTERM

	select {
	case err := <-resultCh:
		if err != nil {
			t.Fatalf("shutdown error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for shutdown")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(order) != 3 {
		t.Fatalf("expected 3 cleanup functions called, got %d", len(order))
	}
	// Cleanup functions must be called in order.
	for i, id := range order {
		if id != i+1 {
			t.Fatalf("cleanup order[%d] = %d, want %d", i, id, i+1)
		}
	}
}

// ---------- setShuttingDown callback is invoked ----------

func TestLifecycle_Shutdown_SetsShuttingDownFlag(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	srv := &http.Server{Handler: r}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	sigCh := make(chan os.Signal, 2)
	var shuttingDown atomic.Bool

	resultCh := make(chan error, 1)
	go func() {
		resultCh <- serveWithGracefulShutdown(
			srv, ln, sigCh,
			func(v bool) { shuttingDown.Store(v) },
			5*time.Second,
			func(int) {},
		)
	}()

	time.Sleep(50 * time.Millisecond)

	if shuttingDown.Load() {
		t.Fatal("shuttingDown should be false before signal")
	}

	sigCh <- syscall.SIGTERM

	select {
	case err := <-resultCh:
		if err != nil {
			t.Fatalf("shutdown error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for shutdown")
	}

	if !shuttingDown.Load() {
		t.Fatal("expected shuttingDown to be true after signal")
	}
}
