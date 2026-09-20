package service

import (
    "context"
    "errors"
    "fmt"
    "log/slog"
    "time"

    "github.com/Garmoro/conversation-summarizer/internal/llm"
    "github.com/Garmoro/conversation-summarizer/internal/model"
    "github.com/Garmoro/conversation-summarizer/internal/repository"
    "github.com/Garmoro/conversation-summarizer/internal/repository/redisrepo"
)

var ErrNotFound = errors.New("summary not found")

type Summarizer struct {
    repository repository.SummaryRepository
    cache repository.Cache
    llm llm.Client
    maxMessages int
    maxContentLength int
    logger *slog.Logger
}

func New(repo repository.SummaryRepository, cache repository.Cache, client llm.Client, maxMessages, maxContentLength int, logger *slog.Logger) *Summarizer {
    if logger == nil { logger = slog.Default() }
    return &Summarizer{repository: repo, cache: cache, llm: client, maxMessages: maxMessages, maxContentLength: maxContentLength, logger: logger}
}

func (s *Summarizer) Create(ctx context.Context, request model.CreateSummaryRequest, force bool) (model.Summary, error) {
    if err := model.ValidateRequest(request); err != nil { return model.Summary{}, fmt.Errorf("validate request: %w", err) }
    if !force {
        cached, err := s.cache.Get(ctx, request.ConversationID)
        if err == nil { s.logger.Info("summary cache hit", "conversation_id", request.ConversationID); return cached, nil }
        if !errors.Is(err, redisrepo.ErrCacheMiss) { s.logger.Warn("redis cache read failed", "conversation_id", request.ConversationID, "error", err) }
        if errors.Is(err, redisrepo.ErrCacheMiss) { s.logger.Info("summary cache miss", "conversation_id", request.ConversationID) }
    }

    limited := model.LimitMessages(request.Messages, s.maxMessages, s.maxContentLength)
    if len(limited) == 0 { return model.Summary{}, fmt.Errorf("conversation contains no usable messages") }
    started := time.Now()
    result, err := s.llm.Summarize(ctx, limited)
    duration := time.Since(started)
    if err != nil { s.logger.Error("LLM request failed", "conversation_id", request.ConversationID, "duration_ms", duration.Milliseconds(), "error", err); return model.Summary{}, fmt.Errorf("summarize conversation: %w", err) }
    if err := result.Validate(); err != nil { s.logger.Error("LLM returned invalid structured result", "conversation_id", request.ConversationID, "error", err); return model.Summary{}, fmt.Errorf("validate LLM result: %w", err) }

    draft := model.SummaryDraft{ConversationID: request.ConversationID, Summary: result.Summary, Problem: result.Problem, ActionsTaken: result.ActionsTaken, Status: result.Status, Sentiment: result.Sentiment, Priority: result.Priority, Topics: result.Topics, Model: result.Model, PromptTokens: result.PromptTokens, CompletionTokens: result.CompletionTokens, LLMDurationMS: duration.Milliseconds(), SourceMessages: limited}
    summary, err := s.repository.Save(ctx, draft)
    if err != nil { s.logger.Error("save summary failed", "conversation_id", request.ConversationID, "error", err); return model.Summary{}, fmt.Errorf("save summary: %w", err) }
    if err := s.cache.Set(ctx, summary); err != nil { s.logger.Warn("cache summary failed", "conversation_id", request.ConversationID, "error", err) }
    s.logger.Info("summary created", "conversation_id", request.ConversationID, "summary_id", summary.ID, "model", summary.Model, "duration_ms", duration.Milliseconds(), "prompt_tokens", summary.PromptTokens, "completion_tokens", summary.CompletionTokens, "total_tokens", summary.PromptTokens+summary.CompletionTokens, "success", true)
    return summary, nil
}

func (s *Summarizer) Latest(ctx context.Context, conversationID int64) (model.Summary, error) {
    cached, err := s.cache.Get(ctx, conversationID)
    if err == nil { s.logger.Info("summary cache hit", "conversation_id", conversationID); return cached, nil }
    if !errors.Is(err, redisrepo.ErrCacheMiss) { s.logger.Warn("redis cache read failed", "conversation_id", conversationID, "error", err) }
    s.logger.Info("summary cache miss", "conversation_id", conversationID)
    summary, err := s.repository.Latest(ctx, conversationID)
    if err != nil { return model.Summary{}, fmt.Errorf("get latest summary: %w", err) }
    if err := s.cache.Set(ctx, summary); err != nil { s.logger.Warn("cache summary failed", "conversation_id", conversationID, "error", err) }
    return summary, nil
}

func (s *Summarizer) History(ctx context.Context, conversationID int64) ([]model.Summary, error) {
    summaries, err := s.repository.History(ctx, conversationID)
    if err != nil { return nil, fmt.Errorf("get summary history: %w", err) }
    return summaries, nil
}
