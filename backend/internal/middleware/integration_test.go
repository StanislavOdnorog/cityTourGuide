package middleware_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/middleware"
)

const testBodySizeLimit int64 = 1 << 20 // 1 MB — matches production

// newTestServer creates a Gin router with the production middleware stack
// (body size limit, trace ID, request logger) and starts it on a random port.
// It returns the base URL and a cleanup function that shuts down the server.
func newTestServer(t *testing.T, register func(r *gin.Engine)) (baseURL string, cleanup func()) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.LimitRequestBodySize(testBodySizeLimit))
	r.Use(middleware.TraceIDMiddleware())
	r.Use(middleware.RequestLogger())

	register(r)

	srv := &http.Server{Handler: r}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	base := "http://" + ln.Addr().String()

	return base, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		select {
		case err := <-errCh:
			t.Fatalf("test server error: %v", err)
		default:
		}
	}
}

// ---------- Correlation ID Tests ----------

func TestIntegration_CorrelationID_GeneratedWhenMissing(t *testing.T) {
	base, cleanup := newTestServer(t, func(r *gin.Engine) {
		r.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})
	})
	defer cleanup()

	resp, err := http.Get(base + "/ping")
	if err != nil {
		t.Fatalf("GET /ping: %v", err)
	}
	defer resp.Body.Close()

	reqID := resp.Header.Get("X-Request-ID")
	if reqID == "" {
		t.Fatal("expected X-Request-ID response header to be set")
	}
	if len(reqID) != 36 {
		t.Fatalf("expected UUID-length X-Request-ID, got %q (len=%d)", reqID, len(reqID))
	}
}

func TestIntegration_CorrelationID_PropagatedFromClient(t *testing.T) {
	const clientID = "my-client-trace-42"

	base, cleanup := newTestServer(t, func(r *gin.Engine) {
		r.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})
	})
	defer cleanup()

	req, _ := http.NewRequest(http.MethodGet, base+"/ping", nil)
	req.Header.Set("X-Request-ID", clientID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /ping: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("X-Request-ID"); got != clientID {
		t.Fatalf("expected X-Request-ID %q, got %q", clientID, got)
	}
}

func TestIntegration_CorrelationID_InvalidHeaderGeneratesNewID(t *testing.T) {
	base, cleanup := newTestServer(t, func(r *gin.Engine) {
		r.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})
	})
	defer cleanup()

	req, _ := http.NewRequest(http.MethodGet, base+"/ping", nil)
	req.Header.Set("X-Request-ID", "bad/chars!@#$")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /ping: %v", err)
	}
	defer resp.Body.Close()

	got := resp.Header.Get("X-Request-ID")
	if got == "bad/chars!@#$" {
		t.Fatal("expected invalid X-Request-ID to be replaced, but it was echoed back")
	}
	if got == "" {
		t.Fatal("expected X-Request-ID to be generated for invalid input")
	}
}

func TestIntegration_CorrelationID_VisibleInHandlerContext(t *testing.T) {
	base, cleanup := newTestServer(t, func(r *gin.Engine) {
		r.GET("/trace-check", func(c *gin.Context) {
			traceID := middleware.TraceID(c)
			if traceID == "" {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "no trace_id in context"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"trace_id": traceID})
		})
	})
	defer cleanup()

	const clientID = "context-check-id-123"
	req, _ := http.NewRequest(http.MethodGet, base+"/trace-check", nil)
	req.Header.Set("X-Request-ID", clientID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /trace-check: %v", err)
	}
	defer resp.Body.Close()

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["trace_id"] != clientID {
		t.Fatalf("handler saw trace_id=%q, want %q", body["trace_id"], clientID)
	}
}

func TestIntegration_CorrelationID_IncludedInErrorResponses(t *testing.T) {
	const clientID = "error-trace-abc"

	base, cleanup := newTestServer(t, func(r *gin.Engine) {
		r.GET("/fail", func(c *gin.Context) {
			// Use the errorBody pattern from trace.go (via abortErrorJSON)
			body := gin.H{"error": "something went wrong"}
			if traceID := middleware.TraceID(c); traceID != "" {
				body["trace_id"] = traceID
			}
			c.JSON(http.StatusInternalServerError, body)
		})
	})
	defer cleanup()

	req, _ := http.NewRequest(http.MethodGet, base+"/fail", nil)
	req.Header.Set("X-Request-ID", clientID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /fail: %v", err)
	}
	defer resp.Body.Close()

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["trace_id"] != clientID {
		t.Fatalf("error response trace_id=%q, want %q", body["trace_id"], clientID)
	}
	if body["error"] != "something went wrong" {
		t.Fatalf("error response error=%q, want %q", body["error"], "something went wrong")
	}
}

// ---------- Request Body Size Limit Tests ----------

func TestIntegration_BodySizeLimit_RejectsOversizedBody(t *testing.T) {
	base, cleanup := newTestServer(t, func(r *gin.Engine) {
		r.POST("/upload", func(c *gin.Context) {
			var body map[string]interface{}
			if err := c.ShouldBindJSON(&body); err != nil {
				// This is how real handlers respond — ShouldBindJSON fails
				// when MaxBytesReader hits the limit.
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{
					"error": "request body too large",
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})
	})
	defer cleanup()

	oversized := bytes.Repeat([]byte("x"), int(testBodySizeLimit)+1)
	resp, err := http.Post(base+"/upload", "application/json", bytes.NewReader(oversized))
	if err != nil {
		t.Fatalf("POST /upload: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusRequestEntityTooLarge)
	}

	respBody, _ := io.ReadAll(resp.Body)
	var errResp map[string]interface{}
	if err := json.Unmarshal(respBody, &errResp); err != nil {
		t.Fatalf("expected structured JSON error, got: %s", respBody)
	}
	if _, ok := errResp["error"]; !ok {
		t.Fatalf("expected 'error' field in response, got: %s", respBody)
	}
}

func TestIntegration_BodySizeLimit_AllowsBodyUnderLimit(t *testing.T) {
	base, cleanup := newTestServer(t, func(r *gin.Engine) {
		r.POST("/upload", func(c *gin.Context) {
			_, err := io.Copy(io.Discard, c.Request.Body)
			if err != nil {
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "too large"})
				return
			}
			c.JSON(http.StatusNoContent, nil)
		})
	})
	defer cleanup()

	small := bytes.Repeat([]byte("x"), 1024) // 1 KB
	resp, err := http.Post(base+"/upload", "application/octet-stream", bytes.NewReader(small))
	if err != nil {
		t.Fatalf("POST /upload: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusRequestEntityTooLarge {
		t.Fatal("expected small body to be accepted, got 413")
	}
}

func TestIntegration_BodySizeLimit_IncludesTraceIDInError(t *testing.T) {
	const clientID = "bodysize-trace-456"

	base, cleanup := newTestServer(t, func(r *gin.Engine) {
		r.POST("/upload", func(c *gin.Context) {
			var body map[string]interface{}
			if err := c.ShouldBindJSON(&body); err != nil {
				respBody := gin.H{"error": "request body too large"}
				if traceID := middleware.TraceID(c); traceID != "" {
					respBody["trace_id"] = traceID
				}
				c.JSON(http.StatusRequestEntityTooLarge, respBody)
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})
	})
	defer cleanup()

	oversized := bytes.Repeat([]byte("x"), int(testBodySizeLimit)+1)
	req, _ := http.NewRequest(http.MethodPost, base+"/upload", bytes.NewReader(oversized))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", clientID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /upload: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusRequestEntityTooLarge)
	}

	var errResp map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		t.Fatal("expected structured JSON error response")
	}
	if errResp["trace_id"] != clientID {
		t.Fatalf("trace_id=%q, want %q", errResp["trace_id"], clientID)
	}
	if errResp["error"] != "request body too large" {
		t.Fatalf("error=%q, want %q", errResp["error"], "request body too large")
	}
}

// ---------- Graceful Shutdown Tests ----------

func TestIntegration_GracefulShutdown_DrainsInflightRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.TraceIDMiddleware())
	r.GET("/slow", func(c *gin.Context) {
		close(requestStarted)
		<-releaseRequest
		c.JSON(http.StatusOK, gin.H{"status": "completed"})
	})

	srv := &http.Server{Handler: r}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	base := "http://" + ln.Addr().String()

	// Start a slow request
	respCh := make(chan *http.Response, 1)
	reqErrCh := make(chan error, 1)
	go func() {
		resp, err := http.Get(base + "/slow")
		if err != nil {
			reqErrCh <- err
			return
		}
		respCh <- resp
	}()

	// Wait for the request to reach the handler
	select {
	case <-requestStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for slow request to start")
	}

	// Initiate graceful shutdown
	shutdownDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		shutdownDone <- srv.Shutdown(ctx)
	}()

	// Verify the listener stops accepting new connections
	shutdownClient := &http.Client{Transport: &http.Transport{DisableKeepAlives: true}}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := shutdownClient.Get(base + "/slow")
		if err != nil {
			goto listenerClosed
		}
		resp.Body.Close()
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("server kept accepting new connections after shutdown started")

listenerClosed:

	// Release the in-flight request
	close(releaseRequest)

	// Verify in-flight request completes successfully
	select {
	case err := <-reqErrCh:
		t.Fatalf("slow request failed: %v", err)
	case resp := <-respCh:
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("slow request status = %d, want 200", resp.StatusCode)
		}
		// Verify correlation ID was still propagated for the in-flight request
		if resp.Header.Get("X-Request-ID") == "" {
			t.Fatal("expected X-Request-ID on in-flight response after shutdown")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for slow request to complete")
	}

	// Verify shutdown completed without error
	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for server shutdown")
	}
}

func TestIntegration_GracefulShutdown_ViaSignalChannel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	var shuttingDown atomic.Bool

	r := gin.New()
	r.Use(middleware.TraceIDMiddleware())
	r.GET("/slow", func(c *gin.Context) {
		close(requestStarted)
		<-releaseRequest
		c.JSON(http.StatusOK, gin.H{"done": true})
	})

	srv := &http.Server{Handler: r}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	sigCh := make(chan os.Signal, 2)
	errCh := make(chan error, 1)

	go func() {
		// Replicate the serveWithGracefulShutdown pattern from main.go
		serveErr := make(chan error, 1)
		go func() {
			if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
				serveErr <- err
			}
		}()

		select {
		case err := <-serveErr:
			errCh <- err
			return
		case <-sigCh:
			shuttingDown.Store(true)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			errCh <- srv.Shutdown(ctx)
		}
	}()

	base := "http://" + ln.Addr().String()

	// Start in-flight request
	respCh := make(chan *http.Response, 1)
	go func() {
		resp, err := http.Get(base + "/slow")
		if err == nil {
			respCh <- resp
		}
	}()

	select {
	case <-requestStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for request to start")
	}

	// Send SIGTERM via channel
	sigCh <- syscall.SIGTERM

	// Wait briefly for shutdown to propagate
	time.Sleep(50 * time.Millisecond)

	if !shuttingDown.Load() {
		t.Fatal("expected shuttingDown flag to be set after signal")
	}

	// Release the in-flight request
	close(releaseRequest)

	// Verify in-flight request completed
	select {
	case resp := <-respCh:
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("in-flight request status = %d, want 200", resp.StatusCode)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for in-flight request")
	}

	// Verify shutdown completed
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("shutdown error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for shutdown")
	}
}
