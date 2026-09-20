package service

import (
    "context"
    "testing"
    "time"

    "github.com/Garmoro/conversation-summarizer/internal/model"
    "github.com/Garmoro/conversation-summarizer/internal/repository/redisrepo"
)

type mockLLM struct { calls int; result model.LLMResult; err error }
func (m *mockLLM) Summarize(context.Context, []model.Message) (model.LLMResult, error) { m.calls++; return m.result, m.err }

type mockRepo struct { saves int; latest model.Summary; history []model.Summary; saveErr error }
func (m *mockRepo) Save(context.Context, model.SummaryDraft) (model.Summary, error) { m.saves++; return m.latest, m.saveErr }
func (m *mockRepo) Latest(context.Context, int64) (model.Summary, error) { return m.latest, nil }
func (m *mockRepo) History(context.Context, int64) ([]model.Summary, error) { return m.history, nil }
func (m *mockRepo) Ping(context.Context) error { return nil }

type mockCache struct { values map[int64]model.Summary; gets int; sets int; getErr error }
func (m *mockCache) Get(_ context.Context, id int64) (model.Summary, error) { m.gets++; if m.getErr != nil { return model.Summary{}, m.getErr }; value, ok := m.values[id]; if !ok { return model.Summary{}, redisrepo.ErrCacheMiss }; return value, nil }
func (m *mockCache) Set(_ context.Context, summary model.Summary) error { m.sets++; if m.values == nil { m.values = map[int64]model.Summary{} }; m.values[summary.ConversationID] = summary; return nil }
func (m *mockCache) Ping(context.Context) error { return nil }

func validRequest() model.CreateSummaryRequest { return model.CreateSummaryRequest{ConversationID: 789, Messages: []model.Message{{ID: 1, Sender: model.SenderUser, Content: "Payment failed", CreatedAt: time.Now()}}} }
func validResult() model.LLMResult { return model.LLMResult{Summary: "Payment failed.", Problem: "Payment problem", ActionsTaken: []string{"Retry"}, Status: model.StatusUnresolved, Sentiment: model.SentimentNegative, Priority: model.PriorityHigh, Topics: []string{"payment"}, Model: "mock"} }

func TestCreateUsesCacheHit(t *testing.T) {
    cached := model.Summary{ID: 1, ConversationID: 789, Summary: "cached"}
    cache := &mockCache{values: map[int64]model.Summary{789: cached}}
    llm := &mockLLM{result: validResult()}
    repo := &mockRepo{}
    got, err := New(repo, cache, llm, 30, 20000, nil).Create(context.Background(), validRequest(), false)
    if err != nil { t.Fatal(err) }
    if got.Summary != "cached" || llm.calls != 0 || repo.saves != 0 { t.Fatalf("expected cache hit, got=%+v calls=%d saves=%d", got, llm.calls, repo.saves) }
}

func TestCreateCacheMissCallsLLMAndPersists(t *testing.T) {
    llm := &mockLLM{result: validResult()}
    repo := &mockRepo{latest: model.Summary{ID: 2, ConversationID: 789, Summary: "new"}}
    cache := &mockCache{}
    _, err := New(repo, cache, llm, 30, 20000, nil).Create(context.Background(), validRequest(), false)
    if err != nil { t.Fatal(err) }
    if llm.calls != 1 || repo.saves != 1 || cache.sets != 1 { t.Fatalf("expected miss flow, calls=%d saves=%d sets=%d", llm.calls, repo.saves, cache.sets) }
}

func TestRegenerateIgnoresCache(t *testing.T) {
    cache := &mockCache{values: map[int64]model.Summary{789: {Summary: "old"}}}
    llm := &mockLLM{result: validResult()}
    repo := &mockRepo{latest: model.Summary{ID: 3, ConversationID: 789, Summary: "new"}}
    _, err := New(repo, cache, llm, 30, 20000, nil).Create(context.Background(), validRequest(), true)
    if err != nil { t.Fatal(err) }
    if llm.calls != 1 || cache.gets != 0 { t.Fatalf("expected forced generation, calls=%d gets=%d", llm.calls, cache.gets) }
}

func TestInvalidLLMResultIsRejected(t *testing.T) {
    result := validResult(); result.Status = "invalid"
    _, err := New(&mockRepo{}, &mockCache{}, &mockLLM{result: result}, 30, 20000, nil).Create(context.Background(), validRequest(), true)
    if err == nil { t.Fatal("expected invalid LLM result error") }
}

func TestHistoryReadsRepository(t *testing.T) {
    repo := &mockRepo{history: []model.Summary{{ID: 1}, {ID: 2}}}
    got, err := New(repo, &mockCache{}, &mockLLM{}, 30, 20000, nil).History(context.Background(), 789)
    if err != nil || len(got) != 2 { t.Fatalf("expected history, got=%+v err=%v", got, err) }
}
