package handler

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "log/slog"
    "net/http"
    "strconv"
    "strings"
    "time"

    "github.com/Garmoro/conversation-summarizer/internal/model"
    "github.com/Garmoro/conversation-summarizer/internal/llm"
    "github.com/Garmoro/conversation-summarizer/internal/service"
)

type Dependencies interface {
    Create(context.Context, model.CreateSummaryRequest, bool) (model.Summary, error)
    Latest(context.Context, int64) (model.Summary, error)
    History(context.Context, int64) ([]model.Summary, error)
}

type HealthChecker interface { Ping(context.Context) error }

type Handler struct { service Dependencies; postgres HealthChecker; redis HealthChecker; logger *slog.Logger }

func New(summaryService Dependencies, postgres, redis HealthChecker, logger *slog.Logger) *Handler {
    if logger == nil { logger = slog.Default() }
    return &Handler{service: summaryService, postgres: postgres, redis: redis, logger: logger}
}

func (h *Handler) Routes() http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        started := time.Now()
        defer func() { h.logger.Info("http request", "method", r.Method, "path", r.URL.Path, "duration_ms", time.Since(started).Milliseconds()) }()
        switch {
        case r.Method == http.MethodGet && r.URL.Path == "/health": h.health(w, r)
        case r.Method == http.MethodGet && r.URL.Path == "/swagger/index.html": serveSwagger(w)
        case r.Method == http.MethodGet && r.URL.Path == "/swagger/openapi.yaml": serveOpenAPI(w)
        case r.Method == http.MethodPost && r.URL.Path == "/api/v1/summaries": h.create(w, r, false)
        case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/summaries/"): h.getByConversation(w, r)
        case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/regenerate"): h.createRegenerate(w, r)
        default: writeError(w, http.StatusNotFound, "route not found")
        }
    })
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request, force bool) {
    var request model.CreateSummaryRequest
    if decodeJSON(w, r, &request) != nil { return }
    summary, err := h.service.Create(r.Context(), request, force)
    if err != nil { writeServiceError(w, err); return }
    writeJSON(w, http.StatusCreated, summary)
}

func (h *Handler) createRegenerate(w http.ResponseWriter, r *http.Request) {
    conversationID, suffix, ok := parseConversationPath(r.URL.Path)
    if !ok || suffix != "/regenerate" { writeError(w, http.StatusNotFound, "route not found"); return }
    var request model.CreateSummaryRequest
    if decodeJSON(w, r, &request) != nil { return }
    if request.ConversationID == 0 { request.ConversationID = conversationID }
    if request.ConversationID != conversationID { writeError(w, http.StatusBadRequest, "conversation_id does not match URL"); return }
    summary, err := h.service.Create(r.Context(), request, true)
    if err != nil { writeServiceError(w, err); return }
    writeJSON(w, http.StatusCreated, summary)
}

func (h *Handler) getByConversation(w http.ResponseWriter, r *http.Request) {
    conversationID, suffix, ok := parseConversationPath(r.URL.Path)
    if !ok { writeError(w, http.StatusBadRequest, "conversation_id must be a positive integer"); return }
    switch suffix {
    case "":
        summary, err := h.service.Latest(r.Context(), conversationID)
        if err != nil { writeServiceError(w, err); return }
        writeJSON(w, http.StatusOK, summary)
    case "/history":
        summaries, err := h.service.History(r.Context(), conversationID)
        if err != nil { writeServiceError(w, err); return }
        writeJSON(w, http.StatusOK, map[string]any{"items": summaries})
    default: writeError(w, http.StatusNotFound, "route not found")
    }
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
    checks := map[string]string{}
    status := http.StatusOK
    if err := h.postgres.Ping(r.Context()); err != nil { checks["postgres"] = "down"; status = http.StatusServiceUnavailable } else { checks["postgres"] = "up" }
    if err := h.redis.Ping(r.Context()); err != nil { checks["redis"] = "down"; status = http.StatusServiceUnavailable } else { checks["redis"] = "up" }
    checks["status"] = map[bool]string{true: "ok", false: "degraded"}[status == http.StatusOK]
    writeJSON(w, status, checks)
}

func parseConversationPath(path string) (int64, string, bool) {
    parts := strings.Split(strings.Trim(path, "/"), "/")
    if len(parts) < 4 || parts[0] != "api" || parts[1] != "v1" || parts[2] != "summaries" { return 0, "", false }
    id, err := strconv.ParseInt(parts[3], 10, 64)
    if err != nil || id <= 0 { return 0, "", false }
    suffix := ""
    if len(parts) > 4 { suffix = "/" + strings.Join(parts[4:], "/") }
    return id, suffix, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
    r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
    decoder := json.NewDecoder(r.Body)
    decoder.DisallowUnknownFields()
    if err := decoder.Decode(target); err != nil { writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON body: %v", err)); return err }
    return nil
}

func writeServiceError(w http.ResponseWriter, err error) {
    status := http.StatusInternalServerError
    if strings.Contains(err.Error(), "validate request") || strings.Contains(err.Error(), "conversation contains") { status = http.StatusBadRequest }
    if errors.Is(err, service.ErrNotFound) || strings.Contains(err.Error(), "not found") { status = http.StatusNotFound }
    if errors.Is(err, llm.ErrUnavailable) { status = http.StatusBadGateway }
    if errors.Is(err, llm.ErrTimeout) { status = http.StatusGatewayTimeout }
    if errors.Is(err, llm.ErrInvalidResponse) { status = http.StatusBadGateway }
    if strings.Contains(err.Error(), "validate LLM result") { status = http.StatusBadGateway }
    writeError(w, status, err.Error())
}
func writeError(w http.ResponseWriter, status int, message string) { writeJSON(w, status, map[string]string{"error": message}) }
func writeJSON(w http.ResponseWriter, status int, payload any) { w.Header().Set("Content-Type", "application/json; charset=utf-8"); w.WriteHeader(status); _ = json.NewEncoder(w).Encode(payload) }

func serveSwagger(w http.ResponseWriter) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    _, _ = w.Write([]byte(          `<!doctype html><html><head><title>Conversation Summarizer API</title><link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css"></head><body><div id="swagger-ui"></div><script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script><script>window.ui=SwaggerUIBundle({url:'/swagger/openapi.yaml',dom_id:'#swagger-ui'});</script></body></html>`))
}

func serveOpenAPI(w http.ResponseWriter) {
    w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
    _, _ = w.Write([]byte(          `openapi: 3.0.3
info:
  title: Conversation Summarizer API
  version: 1.0.0
  description: Generates structured support conversation summaries through an OpenAI-compatible LLM.
servers:
  - url: http://localhost:8091
paths:
  /health:
    get:
      summary: Service health
      responses:
        "200": {description: All dependencies are available}
        "503": {description: At least one dependency is unavailable}
  /api/v1/summaries:
    post:
      summary: Create a conversation summary
      requestBody:
        required: true
        content:
          application/json:
            schema: {$ref: '#/components/schemas/CreateSummaryRequest'}
      responses:
        "201": {description: Summary created, content: {application/json: {schema: {$ref: '#/components/schemas/Summary'}}}}
        "400": {description: Invalid input or invalid LLM result}
        "502": {description: LLM unavailable}
  /api/v1/summaries/{conversation_id}:
    get:
      summary: Get latest summary (Redis first, PostgreSQL fallback)
      parameters: [{in: path, name: conversation_id, required: true, schema: {type: integer, format: int64}}]
      responses:
        "200": {description: Latest summary}
        "404": {description: Summary not found}
  /api/v1/summaries/{conversation_id}/history:
    get:
      summary: Get summary history
      parameters: [{in: path, name: conversation_id, required: true, schema: {type: integer, format: int64}}]
      responses:
        "200": {description: Summary versions}
  /api/v1/summaries/{conversation_id}/regenerate:
    post:
      summary: Force a new LLM generation and preserve the old version
      parameters: [{in: path, name: conversation_id, required: true, schema: {type: integer, format: int64}}]
      requestBody:
        required: true
        content:
          application/json:
            schema: {$ref: '#/components/schemas/CreateSummaryRequest'}
      responses:
        "201": {description: New summary version created}
components:
  schemas:
    Message:
      type: object
      required: [id, sender, content, created_at]
      properties:
        id: {type: integer, format: int64}
        sender: {type: string, enum: [user, operator, bot, system]}
        content: {type: string}
        created_at: {type: string, format: date-time}
    CreateSummaryRequest:
      type: object
      required: [conversation_id, messages]
      properties:
        conversation_id: {type: integer, format: int64}
        messages: {type: array, items: {$ref: '#/components/schemas/Message'}}
    Summary:
      type: object
      properties:
        id: {type: integer, format: int64}
        conversation_id: {type: integer, format: int64}
        summary: {type: string}
        problem: {type: string}
        actions_taken: {type: array, items: {type: string}}
        status: {type: string, enum: [resolved, unresolved, waiting_user, waiting_operator, unknown]}
        sentiment: {type: string, enum: [positive, neutral, negative, angry]}
        priority: {type: string, enum: [low, medium, high, critical]}
        topics: {type: array, items: {type: string}}
        model: {type: string}
        prompt_tokens: {type: integer}
        completion_tokens: {type: integer}
        llm_duration_ms: {type: integer}
        created_at: {type: string, format: date-time}
        updated_at: {type: string, format: date-time}
`))
}
