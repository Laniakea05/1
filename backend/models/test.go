package models

import "time"

type PsychologicalTest struct {
	ID            int       `json:"id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Instructions  string    `json:"instructions"`
	EstimatedTime int       `json:"estimated_time"`
	IsActive      bool      `json:"is_active"`
	CreatedBy     int       `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
	Questions     []TestQuestion `json:"questions,omitempty"`
}

type TestQuestion struct {
	ID           int      `json:"id"`
	TestID       int      `json:"test_id"`
	QuestionText string   `json:"question_text"`
	QuestionType string   `json:"question_type"`
	Options      []string `json:"options"`
	Weight       float64  `json:"weight"`
	OrderIndex   int      `json:"order_index"`
}

type TestResult struct {
	ID             int                    `json:"id"`
	UserID         int                    `json:"user_id"`
	TestID         int                    `json:"test_id"`
	Score          float64                `json:"score"`
	MaxScore       float64                `json:"max_score"`
	StartedAt      time.Time              `json:"started_at"`
	CompletedAt    time.Time              `json:"completed_at"`
	Answers        map[string]interface{} `json:"answers"`
	Interpretation string                 `json:"interpretation"`
}