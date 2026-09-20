package postgres

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "sort"
    "strings"

    "github.com/Garmoro/conversation-summarizer/internal/model"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct { pool *pgxpool.Pool }

func New(ctx context.Context, databaseURL string) (*Repository, error) {
    pool, err := pgxpool.New(ctx, databaseURL)
    if err != nil { return nil, fmt.Errorf("create postgres pool: %w", err) }
    if err := pool.Ping(ctx); err != nil { pool.Close(); return nil, fmt.Errorf("ping postgres: %w", err) }
    return &Repository{pool: pool}, nil
}
func (r *Repository) Close() { r.pool.Close() }
func (r *Repository) Ping(ctx context.Context) error { return r.pool.Ping(ctx) }

func (r *Repository) Save(ctx context.Context, draft model.SummaryDraft) (model.Summary, error) {
    actions, err := json.Marshal(draft.ActionsTaken)
    if err != nil { return model.Summary{}, fmt.Errorf("marshal actions: %w", err) }
    topics, err := json.Marshal(draft.Topics)
    if err != nil { return model.Summary{}, fmt.Errorf("marshal topics: %w", err) }
    var summary model.Summary
    err = r.pool.QueryRow(ctx, `
        INSERT INTO conversation_summaries (conversation_id, summary, problem, actions_taken, status, sentiment, priority, topics, model, prompt_tokens, completion_tokens, llm_duration_ms)
        VALUES ($1, $2, $3, $4::jsonb, $5, $6, $7, $8::jsonb, $9, $10, $11, $12)
        RETURNING id, conversation_id, summary, problem, actions_taken, status, sentiment, priority, topics, model, prompt_tokens, completion_tokens, llm_duration_ms, created_at, updated_at`,
        draft.ConversationID, draft.Summary, draft.Problem, actions, draft.Status, draft.Sentiment, draft.Priority, topics, draft.Model, draft.PromptTokens, draft.CompletionTokens, draft.LLMDurationMS,
    ).Scan(&summary.ID, &summary.ConversationID, &summary.Summary, &summary.Problem, &actions, &summary.Status, &summary.Sentiment, &summary.Priority, &topics, &summary.Model, &summary.PromptTokens, &summary.CompletionTokens, &summary.LLMDurationMS, &summary.CreatedAt, &summary.UpdatedAt)
    if err != nil { return model.Summary{}, fmt.Errorf("insert summary: %w", err) }
    if err := json.Unmarshal(actions, &summary.ActionsTaken); err != nil { return model.Summary{}, fmt.Errorf("decode actions: %w", err) }
    if err := json.Unmarshal(topics, &summary.Topics); err != nil { return model.Summary{}, fmt.Errorf("decode topics: %w", err) }
    return summary, nil
}

func (r *Repository) Latest(ctx context.Context, conversationID int64) (model.Summary, error) {
    return scanSummary(r.pool.QueryRow(ctx, summaryQuery+` WHERE conversation_id = $1 ORDER BY created_at DESC, id DESC LIMIT 1`, conversationID).Scan)
}

func (r *Repository) History(ctx context.Context, conversationID int64) ([]model.Summary, error) {
    rows, err := r.pool.Query(ctx, summaryQuery+` WHERE conversation_id = $1 ORDER BY created_at DESC, id DESC`, conversationID)
    if err != nil { return nil, fmt.Errorf("query summary history: %w", err) }
    defer rows.Close()
    var result []model.Summary
    for rows.Next() {
        summary, scanErr := scanSummary(rows.Scan)
        if scanErr != nil { return nil, scanErr }
        result = append(result, summary)
    }
    if err := rows.Err(); err != nil { return nil, fmt.Errorf("iterate summary history: %w", err) }
    return result, nil
}

const summaryQuery = `SELECT id, conversation_id, summary, problem, actions_taken, status, sentiment, priority, topics, model, prompt_tokens, completion_tokens, llm_duration_ms, created_at, updated_at FROM conversation_summaries`

func scanSummary(scan func(...any) error) (model.Summary, error) {
    var summary model.Summary
    var actions, topics []byte
    if err := scan(&summary.ID, &summary.ConversationID, &summary.Summary, &summary.Problem, &actions, &summary.Status, &summary.Sentiment, &summary.Priority, &topics, &summary.Model, &summary.PromptTokens, &summary.CompletionTokens, &summary.LLMDurationMS, &summary.CreatedAt, &summary.UpdatedAt); err != nil {
        if errors.Is(err, pgx.ErrNoRows) { return model.Summary{}, fmt.Errorf("summary not found") }
        return model.Summary{}, fmt.Errorf("scan summary: %w", err)
    }
    if err := json.Unmarshal(actions, &summary.ActionsTaken); err != nil { return model.Summary{}, fmt.Errorf("decode actions: %w", err) }
    if err := json.Unmarshal(topics, &summary.Topics); err != nil { return model.Summary{}, fmt.Errorf("decode topics: %w", err) }
    return summary, nil
}

func RunMigrations(ctx context.Context, databaseURL, migrationsDir string) error {
    pool, err := pgxpool.New(ctx, databaseURL)
    if err != nil { return fmt.Errorf("create migration pool: %w", err) }
    defer pool.Close()
    if err := pool.Ping(ctx); err != nil { return fmt.Errorf("ping postgres for migrations: %w", err) }
    entries, err := os.ReadDir(migrationsDir)
    if err != nil { return fmt.Errorf("read migrations directory: %w", err) }
    var files []string
    for _, entry := range entries { if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") { files = append(files, entry.Name()) } }
    sort.Strings(files)
    for _, name := range files {
        content, readErr := os.ReadFile(filepath.Join(migrationsDir, name))
        if readErr != nil { return fmt.Errorf("read migration %s: %w", name, readErr) }
        if _, execErr := pool.Exec(ctx, string(content)); execErr != nil { return fmt.Errorf("execute migration %s: %w", name, execErr) }
    }
    return nil
}
