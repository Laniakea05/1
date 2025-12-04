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
			&test.EstimatedTime, &test.PassThreshold, &test.MethodologyType)
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
		&test.EstimatedTime, &test.PassThreshold, &test.MethodologyType)

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

	// Преобразуем map в slice и сортируем по order_index
	var questions []interface{}
	for i := 1; i <= len(questionsMap); i++ {
		for _, question := range questionsMap {
			if question.OrderIndex == i {
				questions = append(questions, question)
				break
			}
		}
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

// УНИВЕРСАЛЬНЫЙ РАСЧЕТ ДЛЯ ВСЕХ ТЕСТОВ
func calculateProfessionalTestScore(answers map[string]interface{}, testID int) (float64, float64, string, string, map[string]float64) {
    log.Printf("=== РАСЧЕТ ТЕСТА %d ===", testID)
    log.Printf("Полученные ответы: %v", answers)
    
    // Получаем тип методики и порог
    var methodologyType string
    var passThreshold float64
    err := database.DB.QueryRow(`
        SELECT methodology_type, pass_threshold 
        FROM psychological_tests WHERE id = $1`, testID).Scan(&methodologyType, &passThreshold)
    if err != nil {
        log.Printf("Ошибка получения информации о тесте: %v", err)
        return 0, 0, "Ошибка загрузки теста", "", nil
    }
    
    log.Printf("Методика: %s, Порог: %.1f%%", methodologyType, passThreshold)
    
    // Выбираем метод расчета
    switch methodologyType {
    case "rigidity_scale":
        return calculateTestScore(answers, testID, passThreshold, 20.0, "ригидности")
    case "willpower_control":
        return calculateTestScore(answers, testID, passThreshold, 50.0, "ВСК")
    case "personality_16pf":
        return calculateTestScore(answers, testID, passThreshold, 60.0, "16PF")
    default:
        return calculateGenericScore(answers, passThreshold)
    }
}

// ОБЩАЯ ФУНКЦИЯ РАСЧЕТА ДЛЯ ВСЕХ ТЕСТОВ - ИСПРАВЛЕННАЯ ВЕРСИЯ
func calculateTestScore(answers map[string]interface{}, testID int, passThreshold float64, 
                       maxPossibleScore float64, testName string) (float64, float64, string, string, map[string]float64) {
    
    totalScore := 0.0
    answeredQuestions := 0
    
    log.Printf("=== РАСЧЕТ ТЕСТА %s ===", testName)
    
    // Пройдемся по всем ответам
    for questionKey, userAnswer := range answers {
        questionOrder, err := strconv.Atoi(questionKey)
        if err != nil {
            continue
        }
        
        selectedOptionID := int(userAnswer.(float64))
        answeredQuestions++
        
        // Получаем вопрос по order_index
        var questionID int
        err = database.DB.QueryRow(`
            SELECT id FROM test_questions 
            WHERE test_id = $1 AND order_index = $2
        `, testID, questionOrder).Scan(&questionID)
        
        if err != nil {
            log.Printf("Вопрос %d: не найден в БД", questionOrder)
            continue
        }
        
        // Получаем выбранный вариант ответа
        var scoreValue int
        var optionText string
        err = database.DB.QueryRow(`
            SELECT score_value, option_text 
            FROM question_options 
            WHERE id = $1
        `, selectedOptionID).Scan(&scoreValue, &optionText)
        
        if err != nil {
            log.Printf("Вопрос %d: вариант с ID=%d не найден в БД!", questionOrder, selectedOptionID)
            continue
        }
        
        log.Printf("Вопрос %d (ID=%d): выбрано '%s' (ID=%d) → %d баллов", 
            questionOrder, questionID, optionText, selectedOptionID, scoreValue)
        
        totalScore += float64(scoreValue)
    }
    
    // Вычисляем процент
    var percentage float64
    if maxPossibleScore > 0 {
        percentage = (totalScore / maxPossibleScore) * 100
    }
    
    isPassed := percentage >= passThreshold
    
    log.Printf("=== ИТОГ теста %s: %.1f/%.1f = %.1f%%, Порог: %.1f%%, Пройден: %v ===",
        testName, totalScore, maxPossibleScore, percentage, passThreshold, isPassed)
    
    // Получаем интерпретацию в зависимости от типа теста
    var interpretation, recommendation string
    switch testName {
    case "ригидности":
        interpretation, recommendation = getRigidityResult(isPassed, percentage)
    case "ВСК":
        interpretation, recommendation = getWillpowerResult(isPassed, percentage)
    case "16PF":
        interpretation, recommendation = getPersonality16PFResult(isPassed, percentage)
    default:
        interpretation, recommendation = getGenericResult(isPassed, percentage)
    }
    
    return totalScore, maxPossibleScore, interpretation, recommendation, nil
}

// ОБЩИЙ РАСЧЕТ
func calculateGenericScore(answers map[string]interface{}, passThreshold float64) (float64, float64, string, string, map[string]float64) {
    totalScore := 50.0
    maxPossibleScore := 100.0
    percentage := 50.0
    isPassed := percentage >= passThreshold
    
    interpretation, recommendation := getGenericResult(isPassed, percentage)
    return totalScore, maxPossibleScore, interpretation, recommendation, nil
}

// ФУНКЦИИ ФОРМИРОВАНИЯ РЕЗУЛЬТАТОВ
func getRigidityResult(isPassed bool, percentage float64) (string, string) {
    if isPassed {
        return "✅ НИЗКИЙ УРОВЕНЬ РИГИДНОСТИ - ПРИГОДЕН", 
               fmt.Sprintf("Результат: %.1f%%. Кандидат демонстрирует хорошую психологическую гибкость, способность адаптироваться к изменениям и переключаться между задачами. Подходит для работы в динамичной среде информационной безопасности.", percentage)
    } else {
        return "❌ ВЫСОКИЙ УРОВЕНЬ РИГИДНОСТИ - НЕ ПРИГОДЕН", 
               fmt.Sprintf("Результат: %.1f%%. Выявлен повышенный уровень ригидности. Кандидат может испытывать трудности с адаптацией к изменениям и переключением между задачами, что критично для работы в ИБ.", percentage)
    }
}

func getWillpowerResult(isPassed bool, percentage float64) (string, string) {
    if isPassed {
        return "✅ ВЫСОКИЙ УРОВЕНЬ ВОЛЕВОГО САМОКОНТРОЛЯ - ПРИГОДЕН", 
               fmt.Sprintf("Результат: %.1f%%. Кандидат демонстрирует развитые волевые качества: настойчивость, самодисциплину, эмоциональную устойчивость. Способен эффективно работать в условиях стресса и неопределенности.", percentage)
    } else {
        return "❌ НИЗКИЙ УРОВЕНЬ ВОЛЕВОГО САМОКОНТРОЛЯ - НЕ ПРИГОДЕН", 
               fmt.Sprintf("Результат: %.1f%%. Выявлены недостатки волевой регуляции: возможны проблемы с самодисциплиной, концентрацией внимания и работой в стрессовых ситуациях.", percentage)
    }
}

func getPersonality16PFResult(isPassed bool, percentage float64) (string, string) {
    if isPassed {
        return "✅ БЛАГОПРИЯТНЫЙ ЛИЧНОСТНЫЙ ПРОФИЛЬ - ПРИГОДЕН", 
               fmt.Sprintf("Результат: %.1f%%. Личностный профиль соответствует требованиям работы в информационной безопасности: эмоциональная стабильность, ответственность, самоконтроль.", percentage)
    } else {
        return "❌ НЕБЛАГОПРИЯТНЫЙ ЛИЧНОСТНЫЙ ПРОФИЛЬ - НЕ ПРИГОДЕН", 
               fmt.Sprintf("Результат: %.1f%%. Личностный профиль не полностью соответствует требованиям работы в ИБ.", percentage)
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

    // Проверяем, существует ли тест
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

    log.Printf("=== ОБРАБОТКА ТЕСТА %d: пользователь=%v, ответов=%d ===", 
        testID, userID, len(submission.Answers))

    // Расчет баллов
    score, maxScore, interpretation, _, scalePercentages := calculateProfessionalTestScore(submission.Answers, testID)

    // Сохраняем результаты
    percentage := (score / maxScore) * 100
    var passThreshold float64
    database.DB.QueryRow("SELECT pass_threshold FROM psychological_tests WHERE id = $1", testID).Scan(&passThreshold)
    isPassed := percentage >= passThreshold

    log.Printf("=== ФИНАЛЬНЫЙ РЕЗУЛЬТАТ: баллы=%.1f/%.1f, процент=%.1f%%, порог=%.1f%%, пройден=%v ===", 
        score, maxScore, percentage, passThreshold, isPassed)

    // Сохраняем в БД
    var resultID int
    err = database.DB.QueryRow(`
        INSERT INTO test_results (user_id, test_id, total_score, max_possible_score, percentage, is_passed, interpretation, completed_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, NOW()) RETURNING id
    `, userID, testID, score, maxScore, percentage, isPassed, interpretation).Scan(&resultID)

    if err != nil {
        log.Printf("ОШИБКА сохранения результата: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения результатов"})
        return
    }

    // Сохраняем ответы пользователя
    for questionKey, userAnswer := range submission.Answers {
        questionOrder, _ := strconv.Atoi(questionKey)
        
        // Получаем реальный ID вопроса по order_index
        var realQuestionID int
        err := database.DB.QueryRow(`
            SELECT id FROM test_questions 
            WHERE test_id = $1 AND order_index = $2
        `, testID, questionOrder).Scan(&realQuestionID)
        
        if err != nil {
            log.Printf("Ошибка поиска вопроса с order_index=%d: %v", questionOrder, err)
            continue
        }
        
        if selectedOptionID, ok := userAnswer.(float64); ok {
            database.DB.Exec(`
                INSERT INTO user_answers (result_id, question_id, option_id)
                VALUES ($1, $2, $3)
            `, resultID, realQuestionID, int(selectedOptionID))
        }
    }

    log.Printf("=== ТЕСТ %d УСПЕШНО СОХРАНЕН: пользователь=%v, пройден=%t ===", testID, userID, isPassed)

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