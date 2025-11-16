package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"psycho-test-system/database"

	"github.com/gin-gonic/gin"
)

func GetUserProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Пользователь не авторизован"})
		return
	}

	var user struct {
		ID       int    `json:"id"`
		Email    string `json:"email"`
		FullName string `json:"full_name"`
		Role     string `json:"role"`
	}

	err := database.DB.QueryRow(`
		SELECT id, email, full_name, role 
		FROM users 
		WHERE id = $1
	`, userID).Scan(&user.ID, &user.Email, &user.FullName, &user.Role)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения данных пользователя"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

func GetUserStats(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Пользователь не авторизован"})
		return
	}

	log.Printf("Getting stats for user ID: %d", userID)

	// Получаем статистику тестов пользователя
	var stats struct {
		TestsCompleted int     `json:"tests_completed"`
		AverageScore   float64 `json:"average_score"`
		LastTestDate   string  `json:"last_test_date"`
	}

	// Количество пройденных тестов
	err := database.DB.QueryRow(`
		SELECT COUNT(*) FROM test_results 
		WHERE user_id = $1
	`, userID).Scan(&stats.TestsCompleted)
	if err != nil {
		log.Printf("Error getting tests count: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения статистики"})
		return
	}

	log.Printf("Tests completed: %d", stats.TestsCompleted)

	// Средний балл
	err = database.DB.QueryRow(`
		SELECT COALESCE(AVG(score), 0) FROM test_results 
		WHERE user_id = $1
	`, userID).Scan(&stats.AverageScore)
	if err != nil {
		log.Printf("Error getting average score: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения среднего балла"})
		return
	}

	log.Printf("Average score: %.2f", stats.AverageScore)

	// Дата последнего теста
	var lastDate sql.NullString
	err = database.DB.QueryRow(`
		SELECT TO_CHAR(completed_at, 'DD.MM.YYYY')
		FROM test_results 
		WHERE user_id = $1 
		ORDER BY completed_at DESC 
		LIMIT 1
	`, userID).Scan(&lastDate)
	
	if err != nil {
		log.Printf("Error getting last test date: %v", err)
		stats.LastTestDate = "-"
	} else if lastDate.Valid {
		stats.LastTestDate = lastDate.String
		log.Printf("Last test date: %s", stats.LastTestDate)
	} else {
		stats.LastTestDate = "-"
		log.Printf("No valid last test date found")
	}

	c.JSON(http.StatusOK, gin.H{
		"stats": stats,
	})
}

func UpdateUserProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Пользователь не авторизован"})
		return
	}

	var updateData struct {
		FullName string `json:"full_name"`
		Email    string `json:"email"`
	}

	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}

	_, err := database.DB.Exec(`
		UPDATE users 
		SET full_name = $1, email = $2 
		WHERE id = $3
	`, updateData.FullName, updateData.Email, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления профиля"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Профиль обновлен"})
}

func CheckDBStructure(c *gin.Context) {
	// Проверим структуру test_results
	rows, err := database.DB.Query(`
		SELECT column_name, data_type, is_nullable 
		FROM information_schema.columns 
		WHERE table_name = 'test_results' 
		ORDER BY ordinal_position
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	
	var columns []map[string]string
	for rows.Next() {
		var colName, dataType, nullable string
		rows.Scan(&colName, &dataType, &nullable)
		columns = append(columns, map[string]string{
			"name": colName,
			"type": dataType,
			"nullable": nullable,
		})
	}
	
	// Проверим есть ли данные в test_results
	var testResultsCount int
	database.DB.QueryRow("SELECT COUNT(*) FROM test_results").Scan(&testResultsCount)
	
	c.JSON(http.StatusOK, gin.H{
		"table_columns": columns,
		"test_results_count": testResultsCount,
	})
}