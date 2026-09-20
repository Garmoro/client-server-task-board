package repository

import (
    "context"
    "github.com/Garmoro/conversation-summarizer/internal/model"
)

type SummaryRepository interface {
    Save(context.Context, model.SummaryDraft) (model.Summary, error)
    Latest(context.Context, int64) (model.Summary, error)
    History(context.Context, int64) ([]model.Summary, error)
    Ping(context.Context) error
}

type Cache interface {
    Get(context.Context, int64) (model.Summary, error)
    Set(context.Context, model.Summary) error
    Ping(context.Context) error
}
