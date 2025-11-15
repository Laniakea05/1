package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"psycho-test-system/database"
	"psycho-test-system/models"

	"github.com/gin-gonic/gin"
)

func GetTests(c *gin.Context) {
	rows, err := database.DB.Query(`
		SELECT id, title, description, instructions, estimated_time 
		FROM psychological_tests 
		WHERE is_active = true
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения тестов"})
		return
	}
	defer rows.Close()

	var tests []models.PsychologicalTest
	for rows.Next() {
		var test models.PsychologicalTest
		err := rows.Scan(&test.ID, &test.Title, &test.Description, &test.Instructions, &test.EstimatedTime)
		if err != nil {
			continue
		}
		tests = append(tests, test)
	}

	c.JSON(http.StatusOK, gin.H{"tests": tests})
}

func GetTest(c *gin.Context) {
	testID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID теста"})
		return
	}

	var test models.PsychologicalTest
	err = database.DB.QueryRow(`
		SELECT id, title, description, instructions, estimated_time 
		FROM psychological_tests 
		WHERE id = $1 AND is_active = true
	`, testID).Scan(&test.ID, &test.Title, &test.Description, &test.Instructions, &test.EstimatedTime)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Тест не найден"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка базы данных"})
		return
	}

	rows, err := database.DB.Query(`
		SELECT id, question_text, question_type, options, weight, order_index
		FROM test_questions 
		WHERE test_id = $1 
		ORDER BY order_index
	`, testID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения вопросов"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var question models.TestQuestion
		var optionsJSON string
		
		err := rows.Scan(&question.ID, &question.QuestionText, &question.QuestionType, &optionsJSON, &question.Weight, &question.OrderIndex)
		if err != nil {
			continue
		}

		json.Unmarshal([]byte(optionsJSON), &question.Options)
		test.Questions = append(test.Questions, question)
	}

	c.JSON(http.StatusOK, gin.H{"test": test})
}

// calculateTestScore вычисляет результат теста на основе ответов
func calculateTestScore(answers map[string]interface{}) (float64, float64, string) {
	// Правильные ответы для теста на стрессоустойчивость
	// question_id -> index_of_correct_answer
	correctAnswers := map[string]int{
		"1": 1, // "Спокойно анализирую ситуацию и действую по инструкции"
		"2": 0, // "Эффективно планируете задачи и распределяете время"
		"3": 0, // "Принимаю к сведению и работаю над ошибками"
	}
	
	totalScore := 0.0
	maxPossibleScore := float64(len(correctAnswers))
	
	// Считаем правильные ответы
	for qID, userAnswer := range answers {
		if correctIndex, exists := correctAnswers[qID]; exists {
			// Приводим userAnswer к int (JSON числа приходят как float64)
			var userAns int
			switch v := userAnswer.(type) {
			case float64:
				userAns = int(v)
			case int:
				userAns = v
			default:
				userAns = -1 // невалидный ответ
			}
			
			if userAns == correctIndex {
				totalScore += 1.0
			}
		}
	}
	
	// Вычисляем процент
	percentage := (totalScore / maxPossibleScore) * 100
	
	// Интерпретация результатов
	var interpretation string
	switch {
	case percentage >= 90:
		interpretation = "🎉 Отличный результат! Вы обладаете высоким уровнем стрессоустойчивости, что критически важно для работы в информационной безопасности."
	case percentage >= 70:
		interpretation = "✅ Хороший уровень стрессоустойчивости. Вы умеете сохранять спокойствие в сложных ситуациях."
	case percentage >= 50:
		interpretation = "⚠️ Средний уровень. В стрессовых ситуациях можете терять эффективность. Рекомендуется развивать навыки управления стрессом."
	default:
		interpretation = "🔴 Требуется развитие стрессоустойчивости. Рекомендуется пройти тренинг по управлению стрессом и развивать навыки работы в нештатных ситуациях."
	}
	
	return percentage, 100.0, interpretation
}

func SubmitTest(c *gin.Context) {
    testID, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID теста"})
        return
    }

    // ДЛЯ ДЕБАГА: логируем сырые данные
    bodyBytes, err := io.ReadAll(c.Request.Body)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Ошибка чтения тела запроса"})
        return
    }
    
    log.Printf("Raw JSON received for test %d: %s", testID, string(bodyBytes))
    
    // Восстанавливаем тело запроса для парсинга
    c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

    var submission struct {
        Answers map[string]interface{} `json:"answers"`
    }

    // Используем BindJSON для лучшего контроля ошибок
    if err := c.BindJSON(&submission); err != nil {
        log.Printf("JSON parsing error for test %d: %v", testID, err)
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Неверный формат JSON данных",
            "details": err.Error(),
        })
        return
    }

    log.Printf("Parsed answers for test %d: %+v", testID, submission.Answers)

    userID, exists := c.Get("userID")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Пользователь не авторизован"})
        return
    }

    answersJSON, err := json.Marshal(submission.Answers)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обработки ответов"})
        return
    }

    // РЕАЛЬНАЯ логика оценки теста
    score, maxScore, interpretation := calculateTestScore(submission.Answers)

    // Сохраняем результаты в базу данных с временем завершения
    _, err = database.DB.Exec(`
        INSERT INTO test_results (user_id, test_id, score, max_score, answers, interpretation, completed_at)
        VALUES ($1, $2, $3, $4, $5, $6, NOW())
    `, userID, testID, score, maxScore, string(answersJSON), interpretation)

    if err != nil {
        log.Printf("Database error for test %d: %v", testID, err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения результатов: " + err.Error()})
        return
    }

    log.Printf("Test result saved successfully: user=%d, test=%d, score=%.1f%%", userID, testID, score)

    c.JSON(http.StatusOK, gin.H{
        "message": "Тест завершен",
        "result": gin.H{
            "score":          score,
            "max_score":      maxScore,
            "interpretation": interpretation,
        },
    })
}