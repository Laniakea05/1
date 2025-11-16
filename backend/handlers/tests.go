package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
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

// calculateTestScoreWithWeights вычисляет результат теста с учетом весов ответов (1-5)
func calculateTestScoreWithWeights(answers map[string]interface{}, testID int) (float64, float64, string, string) {
    rows, err := database.DB.Query(`
        SELECT tq.id, tq.question_text, tq.options
        FROM test_questions tq
        WHERE tq.test_id = $1
        ORDER BY tq.order_index
    `, testID)
    
    if err != nil {
        log.Printf("Error getting test questions: %v", err)
        return 0, 0, "Ошибка загрузки теста", ""
    }
    defer rows.Close()

    totalScore := 0.0
    maxPossibleScore := 0.0
    questionsCount := 0

    for rows.Next() {
        var questionID int
        var questionText string
        var optionsJSON string
        
        err := rows.Scan(&questionID, &questionText, &optionsJSON)
        if err != nil {
            continue
        }

        var options []models.QuestionOption
        json.Unmarshal([]byte(optionsJSON), &options)

        // Считаем максимальный возможный балл для вопроса (максимальный вес 5)
        questionMaxScore := 5.0 // Максимальный вес теперь 5
        maxPossibleScore += questionMaxScore

        // Получаем выбранный ответ пользователя
        if userAnswer, exists := answers[strconv.Itoa(questionID)]; exists {
            if answerIndex, ok := userAnswer.(float64); ok {
                idx := int(answerIndex)
                if idx >= 0 && idx < len(options) {
                    totalScore += options[idx].Weight
                }
            }
        }
        questionsCount++
    }

    // Если нет вопросов, возвращаем ошибку
    if questionsCount == 0 {
        return 0, 0, "Тест не содержит вопросов", "Обратитесь к администратору"
    }

    // Определяем интерпретацию и рекомендацию на основе общего балла
    averageScore := totalScore / float64(questionsCount) // Средний балл за вопрос
    interpretation, recommendation := getInterpretationAndRecommendation(averageScore)

    return totalScore, maxPossibleScore, interpretation, recommendation
}

// getInterpretationAndRecommendation возвращает интерпретацию и рекомендацию по среднему баллу (1-5)
func getInterpretationAndRecommendation(averageScore float64) (string, string) {
    switch {
    case averageScore >= 4.5:
        return "Отличное психологическое состояние", 
               "Вы демонстрируете высокий уровень психологической устойчивости. Продолжайте практиковать здоровые привычки и регулярно отслеживайте свое состояние."
    
    case averageScore >= 3.5:
        return "Хорошее психологическое состояние", 
               "Ваше состояние в норме, но есть потенциал для улучшения. Рекомендуется практиковать техники релаксации и поддерживать work-life баланс."
    
    case averageScore >= 2.5:
        return "Удовлетворительное состояние", 
               "Наблюдается умеренный уровень стресса. Рекомендуется обратить внимание на техники управления стрессом, регулярные перерывы и физическую активность."
    
    case averageScore >= 1.5:
        return "Состояние требует внимания", 
               "Вы испытываете значительный стресс. Рекомендуется консультация психолога, регулярная практика медитации и пересмотр рабочих нагрузок."
    
    default:
        return "Критическое состояние", 
               "Рекомендуется немедленно обратиться к специалисту. Ваше психологическое здоровье требует профессиональной помощи и поддержки."
    }
}



func SubmitTest(c *gin.Context) {
    testID, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID теста"})
        return
    }

    var submission struct {
        Answers map[string]interface{} `json:"answers"`
    }

    if err := c.BindJSON(&submission); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
        return
    }

    userID, exists := c.Get("userID")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Пользователь не авторизован"})
        return
    }

    // Логируем для отладки
    fmt.Printf("Processing test submission: user=%d, test=%d, answers=%d\n", 
        userID, testID, len(submission.Answers))

    // Новая логика оценки теста с весами
    score, maxScore, interpretation, recommendation := calculateTestScoreWithWeights(submission.Answers, testID)

    answersJSON, err := json.Marshal(submission.Answers)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обработки ответов"})
        return
    }

    // Сохраняем результаты в базу данных
    _, err = database.DB.Exec(`
        INSERT INTO test_results (user_id, test_id, score, max_score, answers, interpretation, recommendation, completed_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
    `, userID, testID, score, maxScore, string(answersJSON), interpretation, recommendation)

    if err != nil {
        fmt.Printf("Database error: %v\n", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения результатов"})
        return
    }

    fmt.Printf("Test result saved successfully: user=%d, test=%d\n", userID, testID)

    c.JSON(http.StatusOK, gin.H{
        "message": "Тест завершен",
        "result": gin.H{
            "interpretation": interpretation,
            "recommendation": recommendation,
        },
    })
}