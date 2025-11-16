package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"psycho-test-system/database"
	"psycho-test-system/models"

	"github.com/gin-gonic/gin"
)

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

func UpdateTest(c *gin.Context) {
	testID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID теста"})
		return
	}

	var test models.PsychologicalTest
	if err := c.ShouldBindJSON(&test); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}

	_, err = database.DB.Exec(`
		UPDATE psychological_tests 
		SET title = $1, description = $2, instructions = $3, estimated_time = $4
		WHERE id = $5
	`, test.Title, test.Description, test.Instructions, test.EstimatedTime, testID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления теста"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Тест обновлен"})
}

func DeleteTest(c *gin.Context) {
	testID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID теста"})
		return
	}

	_, err = database.DB.Exec("UPDATE psychological_tests SET is_active = false WHERE id = $1", testID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка удаления теста"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Тест удален"})
}

func GetAllUsers(c *gin.Context) {
	rows, err := database.DB.Query(`
		SELECT id, email, full_name, role, created_at 
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
			ID        int       `json:"id"`
			Email     string    `json:"email"`
			FullName  string    `json:"full_name"`
			Role      string    `json:"role"`
			CreatedAt string    `json:"created_at"`
		}
		
		err := rows.Scan(&user.ID, &user.Email, &user.FullName, &user.Role, &user.CreatedAt)
		if err != nil {
			continue
		}

		users = append(users, map[string]interface{}{
			"id":         user.ID,
			"email":      user.Email,
			"full_name":  user.FullName,
			"role":       user.Role,
			"created_at": user.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

func GetAllTests(c *gin.Context) {
	rows, err := database.DB.Query(`
		SELECT id, title, description, instructions, estimated_time, is_active, created_at
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
		}
		
		err := rows.Scan(&test.ID, &test.Title, &test.Description, &test.Instructions, &test.EstimatedTime, &test.IsActive, &test.CreatedAt)
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
		})
	}

	c.JSON(http.StatusOK, gin.H{"tests": tests})
}