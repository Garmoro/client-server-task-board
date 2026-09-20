CREATE TABLE IF NOT EXISTS conversation_summaries (
    id BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT NOT NULL,
    summary TEXT NOT NULL,
    problem TEXT NOT NULL,
    actions_taken JSONB NOT NULL DEFAULT '[]'::jsonb,
    status TEXT NOT NULL CHECK (status IN ('resolved', 'unresolved', 'waiting_user', 'waiting_operator', 'unknown')),
    sentiment TEXT NOT NULL CHECK (sentiment IN ('positive', 'neutral', 'negative', 'angry')),
    priority TEXT NOT NULL CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    topics JSONB NOT NULL DEFAULT '[]'::jsonb,
    model TEXT NOT NULL,
    prompt_tokens INTEGER NOT NULL DEFAULT 0,
    completion_tokens INTEGER NOT NULL DEFAULT 0,
    llm_duration_ms BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_conversation_summaries_conversation_created
    ON conversation_summaries (conversation_id, created_at DESC, id DESC);
