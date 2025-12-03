
package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"psycho-test-system/database"

	"github.com/gin-gonic/gin"
)

// Структуры для статистики
type AdminStats struct {
	TotalUsers       int               `json:"total_users"`
	TotalTests       int               `json:"total_tests"`
	ActiveToday      int               `json:"active_today"`
	PassedTests      int               `json:"passed_tests"`
	FailedTests      int               `json:"failed_tests"`
	AverageSuccess   string            `json:"average_success"`
	MethodologyStats []MethodologyStat `json:"methodology_stats"`
}

type MethodologyStat struct {
	Methodology string  `json:"methodology"`
	TotalTests  int     `json:"total_tests"`
	PassedTests int     `json:"passed_tests"`
	SuccessRate float64 `json:"success_rate"`
}

// Получение статистики для админ-панели
func GetAdminStats(c *gin.Context) {
	var stats AdminStats

	// Всего пользователей
	err := database.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&stats.TotalUsers)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения статистики пользователей"})
		return
	}

	// Всего пройденных тестов
	err = database.DB.QueryRow("SELECT COUNT(*) FROM test_results").Scan(&stats.TotalTests)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения статистики тестов"})
		return
	}

	// Активных сегодня (пользователей, которые проходили тесты сегодня)
	err = database.DB.QueryRow(`
		SELECT COUNT(DISTINCT user_id) 
		FROM test_results 
		WHERE completed_at >= CURRENT_DATE
	`).Scan(&stats.ActiveToday)
	if err != nil {
		stats.ActiveToday = 0
	}

	// Успешно пройденные тесты
	err = database.DB.QueryRow("SELECT COUNT(*) FROM test_results WHERE is_passed = true").Scan(&stats.PassedTests)
	if err != nil {
		stats.PassedTests = 0
	}

	// Неуспешные тесты
	err = database.DB.QueryRow("SELECT COUNT(*) FROM test_results WHERE is_passed = false").Scan(&stats.FailedTests)
	if err != nil {
		stats.FailedTests = 0
	}

	// Процент успешных тестов
	if stats.TotalTests > 0 {
		successRate := (float64(stats.PassedTests) / float64(stats.TotalTests)) * 100
		stats.AverageSuccess = fmt.Sprintf("%.1f%%", successRate)
	} else {
		stats.AverageSuccess = "0%"
	}

	// Статистика по методикам
	rows, err := database.DB.Query(`
		SELECT pt.methodology_type, 
		       COUNT(*) as total_tests,
		       COUNT(CASE WHEN tr.is_passed = true THEN 1 END) as passed_tests
		FROM test_results tr
		JOIN psychological_tests pt ON tr.test_id = pt.id
		GROUP BY pt.methodology_type
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var stat MethodologyStat
			var total, passed int
			err := rows.Scan(&stat.Methodology, &total, &passed)
			if err == nil {
				stat.TotalTests = total
				stat.PassedTests = passed
				if total > 0 {
					stat.SuccessRate = (float64(passed) / float64(total)) * 100
				}
				stats.MethodologyStats = append(stats.MethodologyStats, stat)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// Получение всех пользователей
func GetAllUsers(c *gin.Context) {
	rows, err := database.DB.Query(`
		SELECT id, email, last_name, first_name, patronymic, role, is_blocked,
		       TO_CHAR(created_at AT TIME ZONE 'Europe/Moscow', 'YYYY.MM.DD HH24.MI.SS') as created_at,
		       (SELECT COUNT(*) FROM test_results WHERE user_id = users.id) as tests_count,
		       (SELECT COUNT(*) FROM test_results WHERE user_id = users.id AND is_passed = true) as passed_tests
		FROM users 
		ORDER BY created_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения пользователей"})
		return
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var user struct {
			ID         int    `json:"id"`
			Email      string `json:"email"`
			LastName   string `json:"last_name"`
			FirstName  string `json:"first_name"`
			Patronymic string `json:"patronymic"`
			Role       string `json:"role"`
			IsBlocked  bool   `json:"is_blocked"`
			CreatedAt  string `json:"created_at"`
			TestsCount int    `json:"tests_count"`
			PassedTests int  `json:"passed_tests"`
		}
		
		err := rows.Scan(&user.ID, &user.Email, &user.LastName, &user.FirstName, &user.Patronymic, 
			&user.Role, &user.IsBlocked, &user.CreatedAt, &user.TestsCount, &user.PassedTests)
		if err != nil {
			continue
		}

		users = append(users, map[string]interface{}{
			"id":           user.ID,
			"email":        user.Email,
			"last_name":    user.LastName,
			"first_name":   user.FirstName,
			"patronymic":   user.Patronymic,
			"full_name":    user.LastName + " " + user.FirstName + " " + user.Patronymic,
			"role":         user.Role,
			"is_blocked":   user.IsBlocked,
			"created_at":   user.CreatedAt,
			"tests_count":  user.TestsCount,
			"passed_tests": user.PassedTests,
			"success_rate": calculateSuccessRate(user.TestsCount, user.PassedTests),
		})
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

func calculateSuccessRate(totalTests, passedTests int) string {
	if totalTests == 0 {
		return "0%"
	}
	return fmt.Sprintf("%.1f%%", (float64(passedTests)/float64(totalTests))*100)
}

// Блокировка пользователя
func BlockUser(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID пользователя"})
		return
	}

	var requestData struct {
		Blocked bool `json:"blocked"`
	}
	
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}

	_, err = database.DB.Exec("UPDATE users SET is_blocked = $1 WHERE id = $2", requestData.Blocked, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка блокировки пользователя: " + err.Error()})
		return
	}

	action := "заблокирован"
	if !requestData.Blocked {
		action = "разблокирован"
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Пользователь " + action})
}

// Получение всех тестов
func GetAllTests(c *gin.Context) {
	rows, err := database.DB.Query(`
		SELECT id, title, description, instructions, estimated_time, pass_threshold, methodology_type, is_active,
		       TO_CHAR(created_at AT TIME ZONE 'Europe/Moscow', 'YYYY.MM.DD HH24.MI.SS') as created_at,
		       (SELECT COUNT(*) FROM test_questions WHERE test_id = psychological_tests.id) as questions_count,
		       (SELECT COUNT(*) FROM test_results WHERE test_id = psychological_tests.id) as results_count,
		       (SELECT COUNT(*) FROM test_results WHERE test_id = psychological_tests.id AND is_passed = true) as passed_count
		FROM psychological_tests 
		ORDER BY created_at DESC
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
			IsActive      bool    `json:"is_active"`
			CreatedAt     string  `json:"created_at"`
			QuestionsCount int    `json:"questions_count"`
			ResultsCount  int     `json:"results_count"`
			PassedCount   int     `json:"passed_count"`
		}
		
		err := rows.Scan(&test.ID, &test.Title, &test.Description, &test.Instructions, 
			&test.EstimatedTime, &test.PassThreshold, &test.MethodologyType, &test.IsActive, &test.CreatedAt, 
			&test.QuestionsCount, &test.ResultsCount, &test.PassedCount)
		if err != nil {
			continue
		}

		successRate := "0%"
		if test.ResultsCount > 0 {
			successRate = fmt.Sprintf("%.1f%%", (float64(test.PassedCount)/float64(test.ResultsCount))*100)
		}

		methodologyLabel := getMethodologyLabel(test.MethodologyType)

		tests = append(tests, map[string]interface{}{
			"id":              test.ID,
			"title":           test.Title,
			"description":     test.Description,
			"instructions":    test.Instructions,
			"estimated_time":  test.EstimatedTime,
			"pass_threshold":  test.PassThreshold,
			"methodology_type": test.MethodologyType,
			"methodology_label": methodologyLabel,
			"is_active":       test.IsActive,
			"created_at":      test.CreatedAt,
			"questions_count": test.QuestionsCount,
			"results_count":   test.ResultsCount,
			"passed_count":    test.PassedCount,
			"success_rate":    successRate,
		})
	}

	c.JSON(http.StatusOK, gin.H{"tests": tests})
}

func getMethodologyLabel(methodology string) string {
	labels := map[string]string{
		"rigidity_scale":   "Методика измерения ригидности",
		"willpower_control": "Опросник волевого самоконтроля",
		"personality_16pf": "16PF личностный опросник",
	}
	if label, exists := labels[methodology]; exists {
		return label
	}
	return methodology
}

// Удаление теста - ИСПРАВЛЕННАЯ ВЕРСИЯ (сохраняет результаты)
func DeleteTest(c *gin.Context) {
	testID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID теста"})
		return
	}

	// Проверяем существование теста
	var exists bool
	err = database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM psychological_tests WHERE id = $1)", testID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Тест не найден"})
		return
	}

	// Начинаем транзакцию
	tx, err := database.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка начала транзакции"})
		return
	}
	defer tx.Rollback()

	// 1. Устанавливаем test_id = NULL в таблице test_results вместо удаления
	_, err = tx.Exec(`
		UPDATE test_results 
		SET test_id = NULL 
		WHERE test_id = $1
	`, testID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления результатов теста: " + err.Error()})
		return
	}

	// 2. Получаем ID всех вопросов этого теста
	rows, err := tx.Query("SELECT id FROM test_questions WHERE test_id = $1", testID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения вопросов теста: " + err.Error()})
		return
	}
	defer rows.Close()

	var questionIDs []int
	for rows.Next() {
		var questionID int
		if err := rows.Scan(&questionID); err == nil {
			questionIDs = append(questionIDs, questionID)
		}
	}

	// 3. Для каждого вопроса устанавливаем option_id = NULL в user_answers
	// Это нужно чтобы обойти ограничение внешнего ключа
	for _, questionID := range questionIDs {
		// Получаем ID всех вариантов ответов для этого вопроса
		optionRows, err := tx.Query("SELECT id FROM question_options WHERE question_id = $1", questionID)
		if err != nil {
			continue
		}
		
		var optionIDs []int
		for optionRows.Next() {
			var optionID int
			if err := optionRows.Scan(&optionID); err == nil {
				optionIDs = append(optionIDs, optionID)
			}
		}
		optionRows.Close()
		
		// Устанавливаем option_id = NULL для всех ответов пользователей
		for _, optionID := range optionIDs {
			_, err = tx.Exec(`
				UPDATE user_answers 
				SET option_id = NULL 
				WHERE question_id = $1 AND option_id = $2
			`, questionID, optionID)
			if err != nil {
				// Игнорируем ошибки если записей нет
				continue
			}
		}
	}

	// 4. Теперь можем безопасно удалить варианты ответов
	_, err = tx.Exec(`
		DELETE FROM question_options 
		WHERE question_id IN (SELECT id FROM test_questions WHERE test_id = $1)
	`, testID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка удаления вариантов ответов: " + err.Error()})
		return
	}

	// 5. Удаляем вопросы теста
	_, err = tx.Exec("DELETE FROM test_questions WHERE test_id = $1", testID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка удаления вопросов теста"})
		return
	}

	// 6. Удаляем сам тест
	_, err = tx.Exec("DELETE FROM psychological_tests WHERE id = $1", testID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка удаления теста"})
		return
	}

	// Коммитим транзакцию
	err = tx.Commit()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка завершения операции"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Тест удален",
		"note": "Результаты тестирования сохранены в истории",
	})
}

// Получение всех результатов тестирования - ИСПРАВЛЕННАЯ ВЕРСИЯ (работает с удаленными тестами)
func GetAllResults(c *gin.Context) {
	rows, err := database.DB.Query(`
		SELECT 
			tr.id, 
			u.last_name, 
			u.first_name, 
			u.patronymic, 
			u.email, 
			COALESCE(pt.title, '[Удаленный тест]') as test_title,
			COALESCE(pt.methodology_type, 'Неизвестно') as methodology_type,
			tr.total_score, 
			tr.max_possible_score, 
			tr.percentage, 
			tr.is_passed, 
			tr.interpretation,
			TO_CHAR(tr.completed_at AT TIME ZONE 'Europe/Moscow', 'YYYY.MM.DD HH24.MI.SS') as completed_at
		FROM test_results tr
		LEFT JOIN users u ON tr.user_id = u.id
		LEFT JOIN psychological_tests pt ON tr.test_id = pt.id
		ORDER BY tr.completed_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения результатов: " + err.Error()})
		return
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var result struct {
			ID            int     `json:"id"`
			LastName      string  `json:"last_name"`
			FirstName     string  `json:"first_name"`
			Patronymic    string  `json:"patronymic"`
			UserEmail     string  `json:"user_email"`
			TestTitle     string  `json:"test_title"`
			MethodologyType string `json:"methodology_type"`
			TotalScore    float64 `json:"total_score"`
			MaxScore      float64 `json:"max_score"`
			Percentage    float64 `json:"percentage"`
			IsPassed      bool    `json:"is_passed"`
			Interpretation string `json:"interpretation"`
			CompletedAt   string  `json:"completed_at"`
		}
		
		err := rows.Scan(&result.ID, &result.LastName, &result.FirstName, &result.Patronymic, 
			&result.UserEmail, &result.TestTitle, &result.MethodologyType, &result.TotalScore, &result.MaxScore, 
			&result.Percentage, &result.IsPassed, &result.Interpretation, &result.CompletedAt)
		if err != nil {
			continue
		}

		status := "❌ Не пригоден"
		statusClass := "state-critical"
		if result.IsPassed {
			status = "✅ Пригоден"
			statusClass = "state-excellent"
		}

		methodologyLabel := getMethodologyLabel(result.MethodologyType)
		if result.TestTitle == "[Удаленный тест]" {
			methodologyLabel = "Удаленный тест"
		}

		results = append(results, map[string]interface{}{
			"id":               result.ID,
			"user_name":        result.LastName + " " + result.FirstName + " " + result.Patronymic,
			"user_email":       result.UserEmail,
			"test_title":       result.TestTitle,
			"methodology_type": result.MethodologyType,
			"methodology_label": methodologyLabel,
			"score":           fmt.Sprintf("%.1f/%.1f", result.TotalScore, result.MaxScore),
			"percentage":      fmt.Sprintf("%.1f%%", result.Percentage),
			"is_passed":       result.IsPassed,
			"status":          status,
			"status_class":    statusClass,
			"interpretation":  result.Interpretation,
			"completed_at":    result.CompletedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

// Создание теста
func CreateTest(c *gin.Context) {
	var createReq struct {
		Title         string `json:"title" binding:"required"`
		Description   string `json:"description"`
		Instructions  string `json:"instructions"`
		EstimatedTime int    `json:"estimated_time"`
		PassThreshold float64 `json:"pass_threshold"`
		MethodologyType string `json:"methodology_type"`
		Questions     []struct {
			QuestionText string `json:"question_text"`
			QuestionType string `json:"question_type"`
			ScaleType    string `json:"scale_type"`
			Weight       float64 `json:"weight"`
			Options      []struct {
				OptionText string `json:"option_text"`
				ScoreValue int    `json:"score_value"`
			} `json:"options"`
		} `json:"questions"`
	}

	if err := c.ShouldBindJSON(&createReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}

	userID, _ := c.Get("userID")
	
	var testID int
	err := database.DB.QueryRow(`
		INSERT INTO psychological_tests (title, description, instructions, estimated_time, pass_threshold, methodology_type, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id
	`, createReq.Title, createReq.Description, createReq.Instructions, createReq.EstimatedTime, createReq.PassThreshold, createReq.MethodologyType, userID).Scan(&testID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания теста"})
		return
	}

	// Сохраняем вопросы теста и варианты ответов
	for i, question := range createReq.Questions {
		var questionID int
		err := database.DB.QueryRow(`
			INSERT INTO test_questions (test_id, question_text, question_type, scale_type, weight, order_index)
			VALUES ($1, $2, $3, $4, $5, $6) RETURNING id
		`, testID, question.QuestionText, question.QuestionType, question.ScaleType, question.Weight, i+1).Scan(&questionID)

		if err != nil {
			continue
		}

		// Сохраняем варианты ответов
		for j, option := range question.Options {
			_, err = database.DB.Exec(`
				INSERT INTO question_options (question_id, option_text, score_value, order_index)
				VALUES ($1, $2, $3, $4)
			`, questionID, option.OptionText, option.ScoreValue, j+1)

			if err != nil {
				continue
			}
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Тест создан",
		"test_id": testID,
	})
}

// Получение полной информации о тесте для редактирования
func GetTestForEdit(c *gin.Context) {
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
        WHERE id = $1
    `, testID).Scan(&test.ID, &test.Title, &test.Description, &test.Instructions, &test.EstimatedTime, &test.PassThreshold, &test.MethodologyType)

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
            OptionText string `json:"option_text"` // Теперь используем option_text для фронтенда
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
                    OptionText string `json:"option_text"`
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
                    OptionText string `json:"option_text"`
                    ScoreValue int    `json:"score_value"`
                    OrderIndex int    `json:"order_index"`
                }{},
            }
        }

        if optionID.Valid {
            questionsMap[questionID].Options = append(questionsMap[questionID].Options, struct {
                ID         int    `json:"id"`
                OptionText string `json:"option_text"`
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

// Обновление теста
func UpdateTest(c *gin.Context) {
	testID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID теста"})
		return
	}

	var updateReq struct {
		Title         string `json:"title" binding:"required"`
		Description   string `json:"description"`
		Instructions  string `json:"instructions"`
		EstimatedTime int    `json:"estimated_time"`
		PassThreshold float64 `json:"pass_threshold"`
		MethodologyType string `json:"methodology_type"`
		Questions     []struct {
			QuestionText string `json:"question_text"`
			QuestionType string `json:"question_type"`
			ScaleType    string `json:"scale_type"`
			Weight       float64 `json:"weight"`
			Options      []struct {
				OptionText string `json:"option_text"`
				ScoreValue int    `json:"score_value"`
			} `json:"options"`
		} `json:"questions"`
	}

	if err := c.ShouldBindJSON(&updateReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}

	// Обновляем основную информацию о тесте
	_, err = database.DB.Exec(`
		UPDATE psychological_tests 
		SET title = $1, description = $2, instructions = $3, estimated_time = $4, pass_threshold = $5, methodology_type = $6
		WHERE id = $7
	`, updateReq.Title, updateReq.Description, updateReq.Instructions, updateReq.EstimatedTime, updateReq.PassThreshold, updateReq.MethodologyType, testID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления теста"})
		return
	}

	// Удаляем старые вопросы и варианты ответов
	_, err = database.DB.Exec("DELETE FROM question_options WHERE question_id IN (SELECT id FROM test_questions WHERE test_id = $1)", testID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления вариантов ответов"})
		return
	}

	_, err = database.DB.Exec("DELETE FROM test_questions WHERE test_id = $1", testID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления вопросов"})
		return
	}

	// Добавляем новые вопросы и варианты ответов
	for i, question := range updateReq.Questions {
		var questionID int
		err := database.DB.QueryRow(`
			INSERT INTO test_questions (test_id, question_text, question_type, scale_type, weight, order_index)
			VALUES ($1, $2, $3, $4, $5, $6) RETURNING id
		`, testID, question.QuestionText, question.QuestionType, question.ScaleType, question.Weight, i+1).Scan(&questionID)

		if err != nil {
			continue
		}

		// Сохраняем варианты ответов
		for j, option := range question.Options {
			_, err = database.DB.Exec(`
				INSERT INTO question_options (question_id, option_text, score_value, order_index)
				VALUES ($1, $2, $3, $4)
			`, questionID, option.OptionText, option.ScoreValue, j+1)

			if err != nil {
				continue
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Тест обновлен"})
}
