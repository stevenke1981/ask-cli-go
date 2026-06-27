package webbridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestServerDeliversAuthenticatedTaskAndResult(t *testing.T) {
	server, err := NewServer("hello from ask", 30*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := server.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = server.Close(context.Background())
	})

	baseURL, token := bridgeCoordinates(t, server.TriggerURL())
	response, err := http.Get(baseURL + "/v1/task?token=" + url.QueryEscape(token))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("task status = %d, body = %s", response.StatusCode, body)
	}

	var task Task
	if err := json.NewDecoder(response.Body).Decode(&task); err != nil {
		t.Fatal(err)
	}
	if task.Prompt != "hello from ask" {
		t.Fatalf("task prompt = %q", task.Prompt)
	}

	claimCtx, claimCancel := context.WithTimeout(context.Background(), time.Second)
	defer claimCancel()
	if err := server.WaitForClaim(claimCtx); err != nil {
		t.Fatalf("WaitForClaim() error = %v", err)
	}

	resultBody, _ := json.Marshal(Result{ID: task.ID, Content: "hello from ChatGPT"})
	resultResponse, err := http.Post(
		baseURL+"/v1/result?token="+url.QueryEscape(token),
		"application/json",
		bytes.NewReader(resultBody),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer resultResponse.Body.Close()
	if resultResponse.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resultResponse.Body)
		t.Fatalf("result status = %d, body = %s", resultResponse.StatusCode, body)
	}

	resultCtx, resultCancel := context.WithTimeout(context.Background(), time.Second)
	defer resultCancel()
	result, err := server.WaitForResult(resultCtx)
	if err != nil {
		t.Fatalf("WaitForResult() error = %v", err)
	}
	if result.Content != "hello from ChatGPT" {
		t.Fatalf("result content = %q", result.Content)
	}
}

func TestServerRejectsWrongToken(t *testing.T) {
	server := startTestServer(t)
	response, err := http.Get(server.baseURL() + "/v1/task?token=wrong")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.StatusCode)
	}
}

func TestServerRejectsWrongResultID(t *testing.T) {
	server := startTestServer(t)
	body, _ := json.Marshal(Result{ID: "wrong", Content: "ignored"})
	response, err := http.Post(
		server.baseURL()+"/v1/result?token="+url.QueryEscape(server.token),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", response.StatusCode)
	}
}

func TestServerWaitsHonorContext(t *testing.T) {
	server := startTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	if err := server.WaitForClaim(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("WaitForClaim() error = %v, want deadline exceeded", err)
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel2()
	if _, err := server.WaitForResult(ctx2); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("WaitForResult() error = %v, want deadline exceeded", err)
	}
}

func startTestServer(t *testing.T) *Server {
	t.Helper()
	server, err := NewServer("test prompt", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = server.Close(context.Background())
	})
	return server
}

func bridgeCoordinates(t *testing.T, triggerURL string) (string, string) {
	t.Helper()
	parsed, err := url.Parse(triggerURL)
	if err != nil {
		t.Fatal(err)
	}
	values, err := url.ParseQuery(parsed.Fragment)
	if err != nil {
		t.Fatal(err)
	}
	port := values.Get("ask-cli-port")
	token := values.Get("ask-cli-token")
	if port == "" || token == "" {
		t.Fatalf("trigger URL lacks bridge coordinates: %s", triggerURL)
	}
	return "http://127.0.0.1:" + port, token
}
