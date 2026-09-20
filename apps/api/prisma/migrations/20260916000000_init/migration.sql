CREATE TABLE "conversations" (
    "id" SERIAL NOT NULL,
    "title" TEXT,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "conversations_pkey" PRIMARY KEY ("id")
);

CREATE TABLE "messages" (
    "id" SERIAL NOT NULL,
    "conversation_id" INTEGER NOT NULL,
    "sender" TEXT NOT NULL,
    "content" TEXT NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "messages_pkey" PRIMARY KEY ("id")
);

CREATE INDEX "messages_conversation_id_created_at_idx" ON "messages"("conversation_id", "created_at");

ALTER TABLE "messages" ADD CONSTRAINT "messages_conversation_id_fkey" FOREIGN KEY ("conversation_id") REFERENCES "conversations"("id") ON DELETE CASCADE ON UPDATE CASCADE;

CREATE TABLE IF NOT EXISTS "conversation_summaries" (
    "id" BIGSERIAL NOT NULL,
    "conversation_id" BIGINT NOT NULL,
    "summary" TEXT NOT NULL,
    "problem" TEXT NOT NULL,
    "actions_taken" JSONB NOT NULL DEFAULT '[]'::jsonb,
    "status" TEXT NOT NULL,
    "sentiment" TEXT NOT NULL,
    "priority" TEXT NOT NULL,
    "topics" JSONB NOT NULL DEFAULT '[]'::jsonb,
    "model" TEXT NOT NULL,
    "prompt_tokens" INTEGER NOT NULL DEFAULT 0,
    "completion_tokens" INTEGER NOT NULL DEFAULT 0,
    "llm_duration_ms" BIGINT NOT NULL DEFAULT 0,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "conversation_summaries_pkey" PRIMARY KEY ("id")
);

CREATE INDEX IF NOT EXISTS "conversation_summaries_conversation_id_created_at_id_idx"
  ON "conversation_summaries"("conversation_id", "created_at", "id");

DO $$
BEGIN
    ALTER TABLE "conversation_summaries"
      ADD CONSTRAINT "conversation_summaries_status_check"
      CHECK ("status" IN ('resolved', 'unresolved', 'waiting_user', 'waiting_operator', 'unknown'));
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    ALTER TABLE "conversation_summaries"
      ADD CONSTRAINT "conversation_summaries_sentiment_check"
      CHECK ("sentiment" IN ('positive', 'neutral', 'negative', 'angry'));
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    ALTER TABLE "conversation_summaries"
      ADD CONSTRAINT "conversation_summaries_priority_check"
      CHECK ("priority" IN ('low', 'medium', 'high', 'critical'));
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
