package model

import (
    "encoding/json"
    "fmt"
    "strings"
    "time"
    "unicode/utf8"
)

type Sender string

const (
    SenderUser     Sender = "user"
    SenderOperator Sender = "operator"
    SenderBot      Sender = "bot"
    SenderSystem   Sender = "system"
)

type Status string

const (
    StatusResolved        Status = "resolved"
    StatusUnresolved      Status = "unresolved"
    StatusWaitingUser     Status = "waiting_user"
    StatusWaitingOperator Status = "waiting_operator"
    StatusUnknown         Status = "unknown"
)

type Sentiment string

const (
    SentimentPositive Sentiment = "positive"
    SentimentNeutral  Sentiment = "neutral"
    SentimentNegative Sentiment = "negative"
    SentimentAngry    Sentiment = "angry"
)

type Priority string

const (
    PriorityLow      Priority = "low"
    PriorityMedium   Priority = "medium"
    PriorityHigh     Priority = "high"
    PriorityCritical Priority = "critical"
)

type Message struct {
    ID        int64     `json:"id"`
    Sender    Sender    `json:"sender"`
    Content   string    `json:"content"`
    CreatedAt time.Time `json:"created_at"`
}

type CreateSummaryRequest struct {
    ConversationID int64     `json:"conversation_id"`
    Messages       []Message `json:"messages"`
}

type Summary struct {
    ID               int64     `json:"id"`
    ConversationID   int64     `json:"conversation_id"`
    Summary          string    `json:"summary"`
    Problem          string    `json:"problem"`
    ActionsTaken     []string  `json:"actions_taken"`
    Status           Status    `json:"status"`
    Sentiment        Sentiment `json:"sentiment"`
    Priority         Priority  `json:"priority"`
    Topics           []string  `json:"topics"`
    Model            string    `json:"model"`
    PromptTokens     int       `json:"prompt_tokens"`
    CompletionTokens int       `json:"completion_tokens"`
    LLMDurationMS    int64     `json:"llm_duration_ms"`
    CreatedAt        time.Time `json:"created_at"`
    UpdatedAt        time.Time `json:"updated_at"`
}

type SummaryDraft struct {
    ConversationID   int64
    Summary          string
    Problem          string
    ActionsTaken     []string
    Status           Status
    Sentiment        Sentiment
    Priority         Priority
    Topics           []string
    Model            string
    PromptTokens     int
    CompletionTokens int
    LLMDurationMS    int64
    SourceMessages   []Message
}

type LLMResult struct {
    Summary      string    `json:"summary"`
    Problem      string    `json:"problem"`
    ActionsTaken []string  `json:"actions_taken"`
    Status       Status    `json:"status"`
    Sentiment    Sentiment `json:"sentiment"`
    Priority     Priority  `json:"priority"`
    Topics       []string  `json:"topics"`
    Model        string
    PromptTokens int
    CompletionTokens int
    Duration     time.Duration
}

// UnmarshalJSON normalizes a common small-model mistake: a string instead of an array.
func (r *LLMResult) UnmarshalJSON(data []byte) error {
    type plain LLMResult
    var raw struct {
        Summary string `json:"summary"`
        Problem string `json:"problem"`
        ActionsTaken json.RawMessage `json:"actions_taken"`
        Status Status `json:"status"`
        Sentiment Sentiment `json:"sentiment"`
        Priority Priority `json:"priority"`
        Topics json.RawMessage `json:"topics"`
    }
    if err := json.Unmarshal(data, &raw); err != nil { return err }
    actions, err := normalizeStringList(raw.ActionsTaken)
    if err != nil { return fmt.Errorf("actions_taken: %w", err) }
    topics, err := normalizeStringList(raw.Topics)
    if err != nil { return fmt.Errorf("topics: %w", err) }
    *r = LLMResult(plain{Summary: raw.Summary, Problem: raw.Problem, ActionsTaken: actions, Status: normalizeStatus(raw.Status), Sentiment: normalizeSentiment(raw.Sentiment), Priority: normalizePriority(raw.Priority), Topics: topics})
    return nil
}

func normalizeStatus(value Status) Status {
    normalized := strings.ToLower(strings.TrimSpace(string(value)))
    switch normalized {
    case "resolved", "solved", "closed", "решено", "решен", "решён", "закрыто", "закрыт", "выполнено": return StatusResolved
    case "unresolved", "open", "pending", "не решено", "нерешено", "проблема не решена", "ожидание решения проблемы": return StatusUnresolved
    case "waiting_user", "waiting for user", "ожидание пользователя", "ждем пользователя", "ждём пользователя": return StatusWaitingUser
    case "waiting_operator", "waiting for operator", "ожидание оператора", "ждем оператора", "ждём оператора": return StatusWaitingOperator
    case "unknown", "неизвестно", "неопределено", "ожидаемый": return StatusUnknown
    default: return StatusUnknown
    }
}

func normalizeSentiment(value Sentiment) Sentiment {
    normalized := strings.ToLower(strings.TrimSpace(string(value)))
    switch normalized {
    case "positive", "положительный", "позитивный", "позитив": return SentimentPositive
    case "neutral", "нейтральный", "нейтрально": return SentimentNeutral
    case "negative", "отрицательный", "негативный", "негатив": return SentimentNegative
    case "angry", "злой", "гневный", "агрессивный": return SentimentAngry
    default: return SentimentNeutral
    }
}

func normalizePriority(value Priority) Priority {
    normalized := strings.ToLower(strings.TrimSpace(string(value)))
    switch normalized {
    case "low", "низкий", "низкая": return PriorityLow
    case "medium", "normal", "средний", "средняя", "обычный": return PriorityMedium
    case "high", "высокий", "высокая": return PriorityHigh
    case "critical", "критический", "критичная", "срочный", "срочная": return PriorityCritical
    default: return PriorityMedium
    }
}

func normalizeStringList(data json.RawMessage) ([]string, error) {
    if len(data) == 0 || string(data) == "null" { return nil, fmt.Errorf("must be an array or string") }
    var list []string
    if err := json.Unmarshal(data, &list); err == nil { return list, nil }
    var item string
    if err := json.Unmarshal(data, &item); err == nil { return []string{item}, nil }
    return nil, fmt.Errorf("must be an array or string")
}

func (r LLMResult) Validate() error {
    if strings.TrimSpace(r.Summary) == "" { return fmt.Errorf("LLM summary is empty") }
    if strings.TrimSpace(r.Problem) == "" { return fmt.Errorf("LLM problem is empty") }
    if !IsValidStatus(r.Status) { return fmt.Errorf("unknown status %q", r.Status) }
    if !IsValidSentiment(r.Sentiment) { return fmt.Errorf("unknown sentiment %q", r.Sentiment) }
    if !IsValidPriority(r.Priority) { return fmt.Errorf("unknown priority %q", r.Priority) }
    if r.ActionsTaken == nil { return fmt.Errorf("LLM actions_taken must be an array") }
    if r.Topics == nil { return fmt.Errorf("LLM topics must be an array") }
    return nil
}

func IsValidStatus(value Status) bool {
    switch value { case StatusResolved, StatusUnresolved, StatusWaitingUser, StatusWaitingOperator, StatusUnknown: return true }; return false
}
func IsValidSentiment(value Sentiment) bool {
    switch value { case SentimentPositive, SentimentNeutral, SentimentNegative, SentimentAngry: return true }; return false
}
func IsValidPriority(value Priority) bool {
    switch value { case PriorityLow, PriorityMedium, PriorityHigh, PriorityCritical: return true }; return false
}

func ValidateRequest(request CreateSummaryRequest) error {
    if request.ConversationID <= 0 { return fmt.Errorf("conversation_id must be positive") }
    if len(request.Messages) == 0 { return fmt.Errorf("messages must not be empty") }
    for index, message := range request.Messages {
        if message.ID <= 0 { return fmt.Errorf("messages[%d].id must be positive", index) }
        switch message.Sender { case SenderUser, SenderOperator, SenderBot, SenderSystem: default: return fmt.Errorf("messages[%d].sender is invalid", index) }
        if strings.TrimSpace(message.Content) == "" { return fmt.Errorf("messages[%d].content must not be empty", index) }
        if message.CreatedAt.IsZero() { return fmt.Errorf("messages[%d].created_at is required", index) }
    }
    return nil
}

func LimitMessages(messages []Message, maxMessages, maxContentLength int) []Message {
    start := 0
    if len(messages) > maxMessages { start = len(messages) - maxMessages }
    limited := append([]Message(nil), messages[start:]...)
    total := 0
    for index := len(limited) - 1; index >= 0; index-- {
        remaining := maxContentLength - total
        if remaining <= 0 { limited[index].Content = ""; continue }
        content := limited[index].Content
        if len(content) > remaining {
            content = content[len(content)-remaining:]
            for !utf8.ValidString(content) { content = content[1:] }
        }
        limited[index].Content = content
        total += len(content)
    }
    output := limited[:0]
    for _, message := range limited { if strings.TrimSpace(message.Content) != "" { output = append(output, message) } }
    return output
}
