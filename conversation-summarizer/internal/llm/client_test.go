package llm

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/http/httptest"
    "sync/atomic"
    "testing"
    "time"

    "github.com/Garmoro/conversation-summarizer/internal/model"
)

func testMessages() []model.Message {
    return []model.Message{{ID: 1, Sender: model.SenderUser, Content: "The payment is missing", CreatedAt: time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)}}
}

func llmEnvelope(content string) map[string]any {
    return map[string]any{"model": "mock-model", "choices": []any{map[string]any{"message": map[string]string{"content": content}}}, "usage": map[string]int{"prompt_tokens": 11, "completion_tokens": 7}}
}

func TestHTTPClientParsesStructuredResponseAndUsage(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(llmEnvelope("{\"summary\":\"Payment issue.\",\"problem\":\"Payment missing\",\"actions_taken\":[],\"status\":\"unresolved\",\"sentiment\":\"negative\",\"priority\":\"high\",\"topics\":[\"payment\"]}"))
    }))
    defer server.Close()
    result, err := NewHTTPClient(server.URL, "test-key", "mock-model", 100, 1, time.Second, nil).Summarize(context.Background(), testMessages())
    if err != nil { t.Fatal(err) }
    if result.Status != model.StatusUnresolved || result.PromptTokens != 11 || result.CompletionTokens != 7 { t.Fatalf("unexpected result: %+v", result) }
}

func TestHTTPClientRejectsInvalidJSON(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _ = json.NewEncoder(w).Encode(llmEnvelope("not-json")) }))
    defer server.Close()
    _, err := NewHTTPClient(server.URL, "", "mock-model", 100, 1, time.Second, nil).Summarize(context.Background(), testMessages())
    if err == nil { t.Fatal("expected invalid JSON error") }
}

func TestHTTPClientRetriesTransientErrors(t *testing.T) {
    var calls atomic.Int32
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if calls.Add(1) < 3 { http.Error(w, "temporary", http.StatusServiceUnavailable); return }
        _ = json.NewEncoder(w).Encode(llmEnvelope("{\"summary\":\"Recovered.\",\"problem\":\"Temporary\",\"actions_taken\":[],\"status\":\"resolved\",\"sentiment\":\"neutral\",\"priority\":\"low\",\"topics\":[]}"))
    }))
    defer server.Close()
    result, err := NewHTTPClient(server.URL, "", "mock-model", 100, 3, time.Second, nil).Summarize(context.Background(), testMessages())
    if err != nil || result.Status != model.StatusResolved || calls.Load() != 3 { t.Fatalf("expected retry success, result=%+v err=%v calls=%d", result, err, calls.Load()) }
}

func TestHTTPClientHonorsTimeout(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { time.Sleep(100 * time.Millisecond); fmt.Fprintln(w, "{}") }))
    defer server.Close()
    ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
    defer cancel()
    _, err := NewHTTPClient(server.URL, "", "mock-model", 100, 1, time.Second, nil).Summarize(ctx, testMessages())
    if err == nil { t.Fatal("expected timeout error") }
}
