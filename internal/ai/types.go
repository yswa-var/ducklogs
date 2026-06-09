package ai

type SQLPlan struct {
	NeedsClarification bool     `json:"needs_clarification"`
	ClarifyingQuestion string   `json:"clarifying_question"`
	Intent             string   `json:"intent"`
	SQL                string   `json:"sql"`
	Explanation        string   `json:"explanation"`
	ReportTitle        string   `json:"report_title"`
	ColumnsExpected    []string `json:"columns_expected"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
}

type ChatResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
}
