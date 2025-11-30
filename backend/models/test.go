
package models

import "time"

type PsychologicalTest struct {
	ID            int       `json:"id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Instructions  string    `json:"instructions"`
	EstimatedTime int       `json:"estimated_time"`
	IsActive      bool      `json:"is_active"`
	PassThreshold float64   `json:"pass_threshold"`
	MethodologyType string  `json:"methodology_type"`
	CreatedBy     int       `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
	Questions     []TestQuestion `json:"questions,omitempty"`
}

type TestQuestion struct {
	ID           int              `json:"id"`
	TestID       int              `json:"test_id"`
	QuestionText string           `json:"question_text"`
	QuestionType string           `json:"question_type"`
	ScaleType    string           `json:"scale_type"`
	Options      []QuestionOption `json:"options"`
	Weight       float64          `json:"weight"`
	OrderIndex   int              `json:"order_index"`
}

type QuestionOption struct {
	ID         int    `json:"id"`
	QuestionID int    `json:"question_id"`
	OptionText string `json:"text"`
	ScoreValue int    `json:"score_value"`
	OrderIndex int    `json:"order_index"`
}

type TestResult struct {
	ID             int       `json:"id"`
	UserID         int       `json:"user_id"`
	TestID         int       `json:"test_id"`
	TotalScore     float64   `json:"total_score"`
	MaxPossibleScore float64 `json:"max_possible_score"`
	Percentage     float64   `json:"percentage"`
	IsPassed       bool      `json:"is_passed"`
	Interpretation string    `json:"interpretation"`
	Recommendation string    `json:"recommendation"`
	ScaleResults   string    `json:"scale_results"` // JSON string with detailed scale results
	StartedAt      time.Time `json:"started_at"`
	CompletedAt    time.Time `json:"completed_at"`
}

type CreateTestRequest struct {
	Title         string        `json:"title" binding:"required"`
	Description   string        `json:"description"`
	Instructions  string        `json:"instructions"`
	EstimatedTime int           `json:"estimated_time"`
	PassThreshold float64       `json:"pass_threshold"`
	MethodologyType string      `json:"methodology_type"`
	Questions     []TestQuestion `json:"questions"`
}

// Структуры для детальных результатов по шкалам
type ScaleResult struct {
	ScaleName  string  `json:"scale_name"`
	Score      float64 `json:"score"`
	MaxScore   float64 `json:"max_score"`
	Percentage float64 `json:"percentage"`
	Interpretation string `json:"interpretation"`
}

type DetailedTestResult struct {
	OverallResult TestResult    `json:"overall_result"`
	ScaleResults  []ScaleResult `json:"scale_results"`
}