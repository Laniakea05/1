
package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"psycho-test-system/database"

	"github.com/gin-gonic/gin"
)

func GetTests(c *gin.Context) {
	rows, err := database.DB.Query(`
		SELECT id, title, description, instructions, estimated_time, pass_threshold, methodology_type
		FROM psychological_tests 
		WHERE is_active = true
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения тестов"})
		return
	}
	defer rows.Close()

	var tests []map[string]interface{}
	for rows.Next() {
		var test struct {
			ID            int     `json:"id"`
			Title         string  `json:"title"`
			Description   string  `json:"description"`
			Instructions  string  `json:"instructions"`
			EstimatedTime int     `json:"estimated_time"`
			PassThreshold float64 `json:"pass_threshold"`
			MethodologyType string `json:"methodology_type"`
		}
		err := rows.Scan(&test.ID, &test.Title, &test.Description, &test.Instructions, 
			&test.EstimatedTime, &test.PassThreshold, &test.MethodologyType) // Исправлено: MethodologyType
		if err != nil {
			continue
		}
		
		tests = append(tests, map[string]interface{}{
			"id":              test.ID,
			"title":           test.Title,
			"description":     test.Description,
			"instructions":    test.Instructions,
			"estimated_time":  test.EstimatedTime,
			"pass_threshold":  test.PassThreshold,
			"methodology_type": test.MethodologyType,
		})
	}

	c.JSON(http.StatusOK, gin.H{"tests": tests})
}

func GetTest(c *gin.Context) {
	testID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID теста"})
		return
	}

	var test struct {
		ID            int     `json:"id"`
		Title         string  `json:"title"`
		Description   string  `json:"description"`
		Instructions  string  `json:"instructions"`
		EstimatedTime int     `json:"estimated_time"`
		PassThreshold float64 `json:"pass_threshold"`
		MethodologyType string `json:"methodology_type"`
	}

	err = database.DB.QueryRow(`
		SELECT id, title, description, instructions, estimated_time, pass_threshold, methodology_type
		FROM psychological_tests 
		WHERE id = $1 AND is_active = true
	`, testID).Scan(&test.ID, &test.Title, &test.Description, &test.Instructions, 
		&test.EstimatedTime, &test.PassThreshold, &test.MethodologyType) // Исправлено: MethodologyType

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Тест не найден"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка базы данных"})
		return
	}

	// Получаем вопросы теста с вариантами ответов
	rows, err := database.DB.Query(`
		SELECT q.id, q.question_text, q.question_type, q.scale_type, q.weight, q.order_index,
		       o.id, o.option_text, o.score_value, o.order_index
		FROM test_questions q
		LEFT JOIN question_options o ON q.id = o.question_id
		WHERE q.test_id = $1 
		ORDER BY q.order_index, o.order_index
	`, testID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения вопросов"})
		return
	}
	defer rows.Close()

	questionsMap := make(map[int]*struct {
		ID           int    `json:"id"`
		QuestionText string `json:"question_text"`
		QuestionType string `json:"question_type"`
		ScaleType    string `json:"scale_type"`
		Weight       float64 `json:"weight"`
		OrderIndex   int    `json:"order_index"`
		Options      []struct {
			ID         int    `json:"id"`
			OptionText string `json:"text"`
			ScoreValue int    `json:"score_value"`
			OrderIndex int    `json:"order_index"`
		} `json:"options"`
	})
	
	for rows.Next() {
		var questionID int
		var questionText, questionType, scaleType string
		var weight float64
		var orderIndex int
		var optionID sql.NullInt64
		var optionText sql.NullString
		var scoreValue sql.NullInt64
		var optionOrder sql.NullInt64

		err := rows.Scan(&questionID, &questionText, &questionType, &scaleType, &weight, &orderIndex,
			&optionID, &optionText, &scoreValue, &optionOrder)
		if err != nil {
			continue
		}

		if _, exists := questionsMap[questionID]; !exists {
			questionsMap[questionID] = &struct {
				ID           int    `json:"id"`
				QuestionText string `json:"question_text"`
				QuestionType string `json:"question_type"`
				ScaleType    string `json:"scale_type"`
				Weight       float64 `json:"weight"`
				OrderIndex   int    `json:"order_index"`
				Options      []struct {
					ID         int    `json:"id"`
					OptionText string `json:"text"`
					ScoreValue int    `json:"score_value"`
					OrderIndex int    `json:"order_index"`
				} `json:"options"`
			}{
				ID:           questionID,
				QuestionText: questionText,
				QuestionType: questionType,
				ScaleType:    scaleType,
				Weight:       weight,
				OrderIndex:   orderIndex,
				Options:      []struct {
					ID         int    `json:"id"`
					OptionText string `json:"text"`
					ScoreValue int    `json:"score_value"`
					OrderIndex int    `json:"order_index"`
				}{},
			}
		}

		if optionID.Valid {
			questionsMap[questionID].Options = append(questionsMap[questionID].Options, struct {
				ID         int    `json:"id"`
				OptionText string `json:"text"`
				ScoreValue int    `json:"score_value"`
				OrderIndex int    `json:"order_index"`
			}{
				ID:         int(optionID.Int64),
				OptionText: optionText.String,
				ScoreValue: int(scoreValue.Int64),
				OrderIndex: int(optionOrder.Int64),
			})
		}
	}

	// Преобразуем map в slice
	var questions []interface{}
	for _, question := range questionsMap {
		questions = append(questions, question)
	}

	c.JSON(http.StatusOK, gin.H{
		"test": map[string]interface{}{
			"id":              test.ID,
			"title":           test.Title,
			"description":     test.Description,
			"instructions":    test.Instructions,
			"estimated_time":  test.EstimatedTime,
			"pass_threshold":  test.PassThreshold,
			"methodology_type": test.MethodologyType,
			"questions":       questions,
		},
	})
}

// calculateProfessionalTestScore вычисляет результат профессионального теста
func calculateProfessionalTestScore(answers map[string]interface{}, testID int) (float64, float64, string, string, map[string]float64) {
    // Получаем информацию о тесте
    var methodologyType string
    var passThreshold float64
    err := database.DB.QueryRow(`
        SELECT methodology_type, pass_threshold 
        FROM psychological_tests WHERE id = $1`, testID).Scan(&methodologyType, &passThreshold)
    if err != nil {
        log.Printf("Error getting test info: %v", err)
        return 0, 0, "Ошибка загрузки теста", "", nil
    }

    // Получаем вопросы и варианты ответов
    rows, err := database.DB.Query(`
        SELECT q.id, q.scale_type, o.id, o.score_value
        FROM test_questions q
        JOIN question_options o ON q.id = o.question_id
        WHERE q.test_id = $1
    `, testID)
    
    if err != nil {
        log.Printf("Error getting questions: %v", err)
        return 0, 0, "Ошибка загрузки теста", "", nil
    }
    defer rows.Close()

    totalScore := 0.0
    maxPossibleScore := 0.0
    scaleScores := make(map[string]float64)
    scaleMaxScores := make(map[string]float64)
    questionScores := make(map[int]int) // question_id -> max_score

    // Сначала собираем информацию о максимальных баллах
    for rows.Next() {
        var questionID int
        var scaleType string
        var optionID int
        var scoreValue int
        
        err := rows.Scan(&questionID, &scaleType, &optionID, &scoreValue)
        if err != nil {
            continue
        }

        // Находим максимальный балл для вопроса
        if currentMax, exists := questionScores[questionID]; !exists || scoreValue > currentMax {
            questionScores[questionID] = scoreValue
        }
    }

    // Вычисляем максимально возможный балл
    for _, maxScore := range questionScores {
        maxPossibleScore += float64(maxScore)
    }

    // Теперь вычисляем баллы пользователя
    for questionIDStr, userAnswer := range answers {
        questionID, err := strconv.Atoi(questionIDStr)
        if err != nil {
            continue
        }

        // Получаем scale_type вопроса
        var scaleType string
        err = database.DB.QueryRow("SELECT scale_type FROM test_questions WHERE id = $1", questionID).Scan(&scaleType)
        if err != nil {
            continue
        }

        // Получаем выбранный вариант ответа
        if selectedOptionID, ok := userAnswer.(float64); ok {
            var userScore int
            err := database.DB.QueryRow(`
                SELECT score_value FROM question_options 
                WHERE id = $1 AND question_id = $2
            `, int(selectedOptionID), questionID).Scan(&userScore)
            
            if err == nil {
                totalScore += float64(userScore)
                scaleScores[scaleType] += float64(userScore)
                scaleMaxScores[scaleType] += float64(questionScores[questionID])
            }
        }
    }

    // Вычисляем процентные результаты по шкалам
    scalePercentages := make(map[string]float64)
    for scaleType, score := range scaleScores {
        if maxScore, exists := scaleMaxScores[scaleType]; exists && maxScore > 0 {
            scalePercentages[scaleType] = (score / maxScore) * 100
        }
    }

    // Определяем общий результат
    percentage := (totalScore / maxPossibleScore) * 100
    isPassed := percentage >= passThreshold

    interpretation, recommendation := getMethodologySpecificResult(methodologyType, isPassed, percentage, scalePercentages)

    return totalScore, maxPossibleScore, interpretation, recommendation, scalePercentages
}

// getMethodologySpecificResult возвращает результат в зависимости от методики
func getMethodologySpecificResult(methodology string, isPassed bool, percentage float64, scaleResults map[string]float64) (string, string) {
    switch methodology {
    case "rigidity_scale":
        return getRigidityResult(isPassed, percentage, scaleResults)
    case "willpower_control":
        return getWillpowerResult(isPassed, percentage, scaleResults)
    case "personality_16pf":
        return getPersonality16PFResult(isPassed, percentage, scaleResults)
    default:
        return getGenericResult(isPassed, percentage)
    }
}

func getRigidityResult(isPassed bool, percentage float64, scaleResults map[string]float64) (string, string) {
    if isPassed {
        return "✅ НИЗКИЙ УРОВЕНЬ РИГИДНОСТИ - ПРИГОДЕН", 
               fmt.Sprintf("Результат: %.1f%%. Кандидат демонстрирует хорошую психологическую гибкость, способность адаптироваться к изменениям и переключаться между задачами. Подходит для работы в динамичной среде информационной безопасности.", percentage)
    } else {
        return "❌ ВЫСОКИЙ УРОВЕНЬ РИГИДНОСТИ - НЕ ПРИГОДЕН", 
               fmt.Sprintf("Результат: %.1f%%. Выявлен повышенный уровень ригидности. Кандидат может испытывать трудности с адаптацией к изменениям и переключением между задачами, что критично для работы в ИБ.", percentage)
    }
}

func getWillpowerResult(isPassed bool, percentage float64, scaleResults map[string]float64) (string, string) {
    if isPassed {
        return "✅ ВЫСОКИЙ УРОВЕНЬ ВОЛЕВОГО САМОКОНТРОЛЯ - ПРИГОДЕН", 
               fmt.Sprintf("Результат: %.1f%%. Кандидат демонстрирует развитые волевые качества: настойчивость, самодисциплину, эмоциональную устойчивость. Способен эффективно работать в условиях стресса и неопределенности.", percentage)
    } else {
        return "❌ НИЗКИЙ УРОВЕНЬ ВОЛЕВОГО САМОКОНТРОЛЯ - НЕ ПРИГОДЕН", 
               fmt.Sprintf("Результат: %.1f%%. Выявлены недостатки волевой регуляции: возможны проблемы с самодисциплиной, концентрацией внимания и работой в стрессовых ситуациях.", percentage)
    }
}

func getPersonality16PFResult(isPassed bool, percentage float64, scaleResults map[string]float64) (string, string) {
    // Анализ ключевых факторов для ИБ специалиста
    var criticalFactors []string
    
    if scaleResults["factor_C"] < 60 {
        criticalFactors = append(criticalFactors, "эмоциональная нестабильность")
    }
    if scaleResults["factor_G"] < 60 {
        criticalFactors = append(criticalFactors, "низкая нормативность поведения")
    }
    if scaleResults["factor_Q3"] < 60 {
        criticalFactors = append(criticalFactors, "низкий самоконтроль")
    }
    if scaleResults["factor_L"] < 40 {
        criticalFactors = append(criticalFactors, "излишняя доверчивость")
    }

    if isPassed && len(criticalFactors) == 0 {
        return "✅ БЛАГОПРИЯТНЫЙ ЛИЧНОСТНЫЙ ПРОФИЛЬ - ПРИГОДЕН", 
               fmt.Sprintf("Результат: %.1f%%. Личностный профиль соответствует требованиям работы в информационной безопасности: эмоциональная стабильность, ответственность, самоконтроль.", percentage)
    } else {
        recommendation := fmt.Sprintf("Результат: %.1f%%. ", percentage)
        if len(criticalFactors) > 0 {
            recommendation += "Выявлены проблемные аспекты: " + joinStrings(criticalFactors, ", ") + ". "
        }
        recommendation += "Личностный профиль не полностью соответствует требованиям работы в ИБ."
        
        return "❌ НЕБЛАГОПРИЯТНЫЙ ЛИЧНОСТНЫЙ ПРОФИЛЬ - НЕ ПРИГОДЕН", recommendation
    }
}

func getGenericResult(isPassed bool, percentage float64) (string, string) {
    if isPassed {
        return "✅ КАНДИДАТ ПРИГОДЕН", 
               fmt.Sprintf("Результат: %.1f%% - превышает пороговое значение. Психологические характеристики соответствуют требованиям должности.", percentage)
    } else {
        return "❌ КАНДИДАТ НЕ ПРИГОДЕН", 
               fmt.Sprintf("Результат: %.1f%% - ниже порогового значения. Психологические характеристики не соответствуют требованиям должности.", percentage)
    }
}

func joinStrings(strings []string, separator string) string {
    if len(strings) == 0 {
        return ""
    }
    if len(strings) == 1 {
        return strings[0]
    }
    result := strings[0]
    for i := 1; i < len(strings); i++ {
        result += separator + strings[i]
    }
    return result
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

    // Проверяем, существует ли тест и активен ли он
    var testTitle string
    var isActive bool
    err = database.DB.QueryRow("SELECT title, is_active FROM psychological_tests WHERE id = $1", testID).Scan(&testTitle, &isActive)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Тест не найден"})
        return
    }
    
    if !isActive {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Тест не доступен для прохождения"})
        return
    }

    // Логируем для отладки
    fmt.Printf("Processing professional test submission: user=%d, test=%d, answers=%d\n", 
        userID, testID, len(submission.Answers))

    // Новая логика оценки профессионального теста
    score, maxScore, interpretation, _, scalePercentages := calculateProfessionalTestScore(submission.Answers, testID)

    // Сохраняем результаты в базу данных
    percentage := (score / maxScore) * 100
    var passThreshold float64
    database.DB.QueryRow("SELECT pass_threshold FROM psychological_tests WHERE id = $1", testID).Scan(&passThreshold)
    isPassed := percentage >= passThreshold

    // Сохраняем основной результат
    var resultID int
    err = database.DB.QueryRow(`
        INSERT INTO test_results (user_id, test_id, total_score, max_possible_score, percentage, is_passed, interpretation, completed_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, NOW()) RETURNING id
    `, userID, testID, score, maxScore, percentage, isPassed, interpretation).Scan(&resultID)

    if err != nil {
        fmt.Printf("Database error: %v\n", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения результатов"})
        return
    }

    // Сохраняем отдельные ответы пользователя
    for questionIDStr, userAnswer := range submission.Answers {
        questionID, err := strconv.Atoi(questionIDStr)
        if err != nil {
            continue
        }

        if selectedOptionID, ok := userAnswer.(float64); ok {
            _, err = database.DB.Exec(`
                INSERT INTO user_answers (result_id, question_id, option_id)
                VALUES ($1, $2, $3)
            `, resultID, questionID, int(selectedOptionID))
            
            if err != nil {
                fmt.Printf("Error saving user answer: %v\n", err)
            }
        }
    }

    fmt.Printf("Professional test result saved successfully: user=%d, test=%d, passed=%t\n", userID, testID, isPassed)

    c.JSON(http.StatusOK, gin.H{
        "message": "Тест завершен",
        "result": gin.H{
            "is_passed":      isPassed,
            "interpretation": interpretation,
            "test_title":     testTitle,
            "scale_results":  scalePercentages,
        },
    })
}
