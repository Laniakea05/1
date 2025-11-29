package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"psycho-test-system/database"
	"psycho-test-system/models"

	"github.com/gin-gonic/gin"
)

// Структуры для статистики
type AdminStats struct {
	TotalUsers    int    `json:"total_users"`
	TotalTests    int    `json:"total_tests"`
	ActiveToday   int    `json:"active_today"`
	AverageState  string `json:"average_state"`
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

	// Среднее состояние на основе интерпретаций
	var avgScore sql.NullFloat64
	err = database.DB.QueryRow(`
		SELECT AVG((score/max_score) * 5) 
		FROM test_results 
		WHERE completed_at >= NOW() - INTERVAL '30 days'
	`).Scan(&avgScore)

	if err != nil || !avgScore.Valid {
		stats.AverageState = "Недостаточно данных"
	} else {
		stats.AverageState = getStateFromScore(avgScore.Float64)
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// Вспомогательная функция для определения состояния по баллу
func getStateFromScore(score float64) string {
	switch {
	case score >= 4.5:
		return "Отличное психологическое состояние"
	case score >= 3.5:
		return "Хорошее психологическое состояние"
	case score >= 2.5:
		return "Удовлетворительное состояние"
	case score >= 1.5:
		return "Состояние требует внимания"
	default:
		return "Критическое состояние"
	}
}

// Получение всех пользователей с правильным форматом даты
func GetAllUsers(c *gin.Context) {
	rows, err := database.DB.Query(`
		SELECT id, email, last_name, first_name, patronymic, role, is_blocked,
		       TO_CHAR(created_at AT TIME ZONE 'Europe/Moscow', 'YYYY.MM.DD HH24.MI.SS') as created_at,
		       (SELECT COUNT(*) FROM test_results WHERE user_id = users.id) as tests_count
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
		}
		
		err := rows.Scan(&user.ID, &user.Email, &user.LastName, &user.FirstName, &user.Patronymic, &user.Role, &user.IsBlocked, &user.CreatedAt, &user.TestsCount)
		if err != nil {
			continue
		}

		users = append(users, map[string]interface{}{
			"id":          user.ID,
			"email":       user.Email,
			"last_name":   user.LastName,
			"first_name":  user.FirstName,
			"patronymic":  user.Patronymic,
			"full_name":   user.LastName + " " + user.FirstName + " " + user.Patronymic,
			"role":        user.Role,
			"is_blocked":  user.IsBlocked,
			"created_at":  user.CreatedAt,
			"tests_count": user.TestsCount,
		})
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
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

	// Блокируем/разблокируем пользователя
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

// Получение всех тестов с правильным форматом даты
func GetAllTests(c *gin.Context) {
	rows, err := database.DB.Query(`
		SELECT id, title, description, instructions, estimated_time, is_active,
		       TO_CHAR(created_at AT TIME ZONE 'Europe/Moscow', 'YYYY.MM.DD HH24.MI.SS') as created_at,
		       (SELECT COUNT(*) FROM test_questions WHERE test_id = psychological_tests.id) as questions_count
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
			ID            int    `json:"id"`
			Title         string `json:"title"`
			Description   string `json:"description"`
			Instructions  string `json:"instructions"`
			EstimatedTime int    `json:"estimated_time"`
			IsActive      bool   `json:"is_active"`
			CreatedAt     string `json:"created_at"`
			QuestionsCount int   `json:"questions_count"`
		}
		
		err := rows.Scan(&test.ID, &test.Title, &test.Description, &test.Instructions, 
			&test.EstimatedTime, &test.IsActive, &test.CreatedAt, &test.QuestionsCount)
		if err != nil {
			continue
		}

		tests = append(tests, map[string]interface{}{
			"id":             test.ID,
			"title":          test.Title,
			"description":    test.Description,
			"instructions":   test.Instructions,
			"estimated_time": test.EstimatedTime,
			"is_active":      test.IsActive,
			"created_at":     test.CreatedAt,
			"questions_count": test.QuestionsCount,
		})
	}

	c.JSON(http.StatusOK, gin.H{"tests": tests})
}

// Удаление теста
func DeleteTest(c *gin.Context) {
	testID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID теста"})
		return
	}

	// Сначала удаляем вопросы теста
	_, err = database.DB.Exec("DELETE FROM test_questions WHERE test_id = $1", testID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка удаления вопросов теста"})
		return
	}

	// Затем удаляем результаты теста
	_, err = database.DB.Exec("DELETE FROM test_results WHERE test_id = $1", testID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка удаления результатов теста"})
		return
	}

	// Удаляем сам тест
	_, err = database.DB.Exec("DELETE FROM psychological_tests WHERE id = $1", testID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка удаления теста"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Тест удален"})
}

// Получение всех результатов тестирования с правильным форматом даты
func GetAllResults(c *gin.Context) {
	rows, err := database.DB.Query(`
		SELECT tr.id, u.last_name, u.first_name, u.patronymic, u.email, pt.title, 
		       tr.score, tr.max_score, tr.interpretation, 
		       TO_CHAR(tr.completed_at AT TIME ZONE 'Europe/Moscow', 'YYYY.MM.DD HH24.MI.SS') as completed_at
		FROM test_results tr
		JOIN users u ON tr.user_id = u.id
		JOIN psychological_tests pt ON tr.test_id = pt.id
		ORDER BY tr.completed_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения результатов"})
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
			Score         float64 `json:"score"`
			MaxScore      float64 `json:"max_score"`
			Interpretation string `json:"interpretation"`
			CompletedAt   string  `json:"completed_at"`
		}
		
		err := rows.Scan(&result.ID, &result.LastName, &result.FirstName, &result.Patronymic, &result.UserEmail, &result.TestTitle,
			&result.Score, &result.MaxScore, &result.Interpretation, &result.CompletedAt)
		if err != nil {
			continue
		}

		percentage := (result.Score / result.MaxScore) * 100

		results = append(results, map[string]interface{}{
			"id":            result.ID,
			"user_name":     result.LastName + " " + result.FirstName + " " + result.Patronymic,
			"user_email":    result.UserEmail,
			"test_title":    result.TestTitle,
			"score":         fmt.Sprintf("%.1f/%.1f", result.Score, result.MaxScore),
			"percentage":    fmt.Sprintf("%.1f%%", percentage),
			"interpretation": result.Interpretation,
			"completed_at":  result.CompletedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

// Создание теста
func CreateTest(c *gin.Context) {
	var createReq models.CreateTestRequest
	if err := c.ShouldBindJSON(&createReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}

	userID, _ := c.Get("userID")
	
	var testID int
	err := database.DB.QueryRow(`
		INSERT INTO psychological_tests (title, description, instructions, estimated_time, created_by)
		VALUES ($1, $2, $3, $4, $5) RETURNING id
	`, createReq.Title, createReq.Description, createReq.Instructions, createReq.EstimatedTime, userID).Scan(&testID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания теста"})
		return
	}

	// Сохраняем вопросы теста
	for i, question := range createReq.Questions {
		optionsJSON, err := json.Marshal(question.Options)
		if err != nil {
			continue
		}

		_, err = database.DB.Exec(`
			INSERT INTO test_questions (test_id, question_text, question_type, options, weight, order_index)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, testID, question.QuestionText, question.QuestionType, string(optionsJSON), question.Weight, i+1)

		if err != nil {
			continue
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

	var test models.PsychologicalTest
	err = database.DB.QueryRow(`
		SELECT id, title, description, instructions, estimated_time
		FROM psychological_tests 
		WHERE id = $1
	`, testID).Scan(&test.ID, &test.Title, &test.Description, &test.Instructions, &test.EstimatedTime)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Тест не найден"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка базы данных"})
		return
	}

	// Получаем вопросы теста
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

// Обновление теста
func UpdateTest(c *gin.Context) {
	testID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID теста"})
		return
	}

	var updateReq models.CreateTestRequest
	if err := c.ShouldBindJSON(&updateReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}

	// Обновляем основную информацию о тесте
	_, err = database.DB.Exec(`
		UPDATE psychological_tests 
		SET title = $1, description = $2, instructions = $3, estimated_time = $4
		WHERE id = $5
	`, updateReq.Title, updateReq.Description, updateReq.Instructions, updateReq.EstimatedTime, testID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления теста"})
		return
	}

	// Удаляем старые вопросы
	_, err = database.DB.Exec("DELETE FROM test_questions WHERE test_id = $1", testID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления вопросов"})
		return
	}

	// Добавляем новые вопросы
	for i, question := range updateReq.Questions {
		optionsJSON, err := json.Marshal(question.Options)
		if err != nil {
			continue
		}

		_, err = database.DB.Exec(`
			INSERT INTO test_questions (test_id, question_text, question_type, options, weight, order_index)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, testID, question.QuestionText, question.QuestionType, string(optionsJSON), question.Weight, i+1)

		if err != nil {
			continue
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Тест обновлен"})
}