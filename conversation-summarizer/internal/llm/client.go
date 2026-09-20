package llm

import (
    "bytes"
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "log/slog"
    "math"
    "net/http"
    "strings"
    "time"

    "github.com/Garmoro/conversation-summarizer/internal/model"
)

var ErrUnavailable = errors.New("LLM unavailable")
var ErrTimeout = errors.New("LLM timeout")
var ErrInvalidResponse = errors.New("invalid LLM response")

type Client interface {
    Summarize(ctx context.Context, messages []model.Message) (model.LLMResult, error)
}

type HTTPClient struct {
    baseURL string
    apiKey string
    model string
    maxTokens int
    maxRetries int
    httpClient *http.Client
    logger *slog.Logger
}

func NewHTTPClient(baseURL, apiKey, modelName string, maxTokens, maxRetries int, timeout time.Duration, logger *slog.Logger) *HTTPClient {
    if logger == nil { logger = slog.Default() }
    return &HTTPClient{baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey, model: modelName, maxTokens: maxTokens, maxRetries: maxRetries, httpClient: &http.Client{Timeout: timeout}, logger: logger}
}

type chatRequest struct {
    Model string `json:"model"`
    Messages []chatMessage `json:"messages"`
    Temperature float64 `json:"temperature"`
    MaxTokens int `json:"max_tokens"`
    ResponseFormat responseFormat `json:"response_format"`
    Think *bool `json:"think,omitempty"`
}

type chatMessage struct { Role string `json:"role"`; Content string `json:"content"` }
type responseFormat struct { Type string `json:"type"` }

type chatResponse struct {
    Model string `json:"model"`
    Choices []struct { Message chatMessage `json:"message"` } `json:"choices"`
    Usage struct { PromptTokens int `json:"prompt_tokens"`; CompletionTokens int `json:"completion_tokens"` } `json:"usage"`
}

func (c *HTTPClient) Summarize(ctx context.Context, messages []model.Message) (model.LLMResult, error) {
    requestStarted := time.Now()
    think := false
    body, err := json.Marshal(chatRequest{Model: c.model, Messages: []chatMessage{{Role: "system", Content: systemPrompt}, {Role: "user", Content: buildPrompt(messages)}}, Temperature: 0.1, MaxTokens: c.maxTokens, ResponseFormat: responseFormat{Type: "json_object"}, Think: &think})
    if err != nil { return model.LLMResult{}, fmt.Errorf("marshal LLM request: %w", err) }
    var lastErr error
    for attempt := 1; attempt <= c.maxRetries; attempt++ {
        started := time.Now()
        result, retry, callErr := c.call(ctx, body)
        if callErr == nil {
            result.Model = c.model
            result.Duration = time.Since(started)
            c.logger.Info("llm request", "model", result.Model, "duration_ms", time.Since(requestStarted).Milliseconds(), "prompt_tokens", result.PromptTokens, "completion_tokens", result.CompletionTokens, "total_tokens", result.PromptTokens+result.CompletionTokens, "success", true)
            return result, nil
        }
        if ctx.Err() != nil {
            if errors.Is(ctx.Err(), context.DeadlineExceeded) { return model.LLMResult{}, fmt.Errorf("%w: %v", ErrTimeout, ctx.Err()) }
            return model.LLMResult{}, ctx.Err()
        }
        lastErr = callErr
        if !retry || attempt == c.maxRetries { break }
        delay := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
        c.logger.Warn("llm request retry", "attempt", attempt, "next_attempt", attempt+1, "delay_ms", delay.Milliseconds(), "error", callErr)
        timer := time.NewTimer(delay)
        select { case <-ctx.Done(): timer.Stop(); return model.LLMResult{}, ctx.Err(); case <-timer.C: }
    }
    c.logger.Error("llm request", "model", c.model, "duration_ms", time.Since(requestStarted).Milliseconds(), "prompt_tokens", 0, "completion_tokens", 0, "total_tokens", 0, "success", false, "error", lastErr)
    return model.LLMResult{}, fmt.Errorf("LLM request failed after %d attempts: %w", c.maxRetries, lastErr)
}

func (c *HTTPClient) call(ctx context.Context, body []byte) (model.LLMResult, bool, error) {
    request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
    if err != nil { return model.LLMResult{}, false, fmt.Errorf("create LLM request: %w", err) }
    request.Header.Set("Content-Type", "application/json")
    if c.apiKey != "" { request.Header.Set("Authorization", "Bearer "+c.apiKey) }
    response, err := c.httpClient.Do(request)
    if err != nil {
        if errors.Is(err, context.DeadlineExceeded) { return model.LLMResult{}, true, fmt.Errorf("%w: %v", ErrTimeout, err) }
        return model.LLMResult{}, true, fmt.Errorf("%w: %v", ErrUnavailable, err)
    }
    defer response.Body.Close()
    responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, 2<<20))
    if readErr != nil { return model.LLMResult{}, true, fmt.Errorf("%w: read LLM response: %v", ErrUnavailable, readErr) }
    if response.StatusCode < 200 || response.StatusCode >= 300 {
        retry := response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500
        if retry { return model.LLMResult{}, true, fmt.Errorf("%w: LLM returned HTTP %d", ErrUnavailable, response.StatusCode) }
        return model.LLMResult{}, false, fmt.Errorf("LLM returned HTTP %d", response.StatusCode)
    }
    var envelope chatResponse
    if err := json.Unmarshal(responseBody, &envelope); err != nil { return model.LLMResult{}, false, fmt.Errorf("%w: decode LLM envelope: %v", ErrInvalidResponse, err) }
    if len(envelope.Choices) == 0 || strings.TrimSpace(envelope.Choices[0].Message.Content) == "" { return model.LLMResult{}, false, fmt.Errorf("%w: LLM returned an empty answer", ErrInvalidResponse) }
    var result model.LLMResult
    if err := json.Unmarshal([]byte(cleanJSON(envelope.Choices[0].Message.Content)), &result); err != nil { return model.LLMResult{}, false, fmt.Errorf("%w: decode structured LLM answer: %v", ErrInvalidResponse, err) }
    result.Model = envelope.Model
    result.PromptTokens = envelope.Usage.PromptTokens
    result.CompletionTokens = envelope.Usage.CompletionTokens
    return result, false, nil
}

const systemPrompt = "You are a support conversation analyst. Return only a valid JSON object. Fields: summary string, problem string, actions_taken array of strings, status, sentiment, priority, topics array of strings. IMPORTANT: status must be exactly one of resolved, unresolved, waiting_user, waiting_operator, unknown; sentiment must be exactly one of positive, neutral, negative, angry; priority must be exactly one of low, medium, high, critical. Use lowercase English enum values only. Do not translate enum values. Do not include markdown or commentary."

func buildPrompt(messages []model.Message) string {
    var builder strings.Builder
    for _, message := range messages {
        fmt.Fprintf(&builder, "[%s] %s: %s", message.CreatedAt.UTC().Format(time.RFC3339), message.Sender, message.Content)
        builder.WriteByte('\n')
    }
    return builder.String()
}

func cleanJSON(content string) string {
    content = strings.TrimSpace(content)
    fence := string([]byte{96, 96, 96})
    content = strings.TrimPrefix(content, fence+"json")
    content = strings.TrimPrefix(content, fence)
    content = strings.TrimSuffix(content, fence)
    return strings.TrimSpace(content)
}
