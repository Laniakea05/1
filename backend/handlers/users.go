package handlers

import (
	"database/sql"
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

	var stats struct {
		TestsCompleted  int    `json:"tests_completed"`
		LastTestDate    string `json:"last_test_date"`
		Recommendation  string `json:"recommendation"` // Заменяем средний балл на рекомендацию
	}

	// Количество пройденных тестов за последний месяц
	err := database.DB.QueryRow(`
		SELECT COUNT(*) FROM test_results 
		WHERE user_id = $1 AND completed_at >= NOW() - INTERVAL '30 days'
	`, userID).Scan(&stats.TestsCompleted)
	if err != nil {
		stats.TestsCompleted = 0
	}

	// Дата последнего теста
	var lastDate sql.NullString
	err = database.DB.QueryRow(`
		SELECT TO_CHAR(completed_at, 'DD.MM.YYYY')
		FROM test_results 
		WHERE user_id = $1 
		ORDER BY completed_at DESC 
		LIMIT 1
	`, userID).Scan(&lastDate)
	
	if err != nil || !lastDate.Valid {
		stats.LastTestDate = "-"
	} else {
		stats.LastTestDate = lastDate.String
	}

	// Получаем рекомендацию на основе среднего балла за последний месяц
	stats.Recommendation = getMonthlyRecommendation(userID.(int))

	c.JSON(http.StatusOK, gin.H{
		"stats": stats,
	})
}

// getMonthlyRecommendation возвращает рекомендацию на основе среднего балла за последний месяц
func getMonthlyRecommendation(userID int) string {
	var avgScore sql.NullFloat64
	
	// Получаем средний балл за все тесты за последний месяц (нормализованный к шкале 1-5)
	err := database.DB.QueryRow(`
		SELECT AVG((score/max_score) * 5)
		FROM test_results 
		WHERE user_id = $1 AND completed_at >= NOW() - INTERVAL '30 days'
	`, userID).Scan(&avgScore)

	if err != nil || !avgScore.Valid {
		return "Пройдите первый тест для получения рекомендаций"
	}

	// Определяем рекомендацию на основе среднего балла
	switch {
	case avgScore.Float64 >= 4.5:
		return "Отличные результаты! Ваше психологическое состояние стабильно на высоком уровне. Продолжайте следить за балансом работы и отдыха."
	
	case avgScore.Float64 >= 3.5:
		return "Хорошие показатели. Рекомендуется поддерживать текущие практики и обращать внимание на периоды повышенного стресса."
	
	case avgScore.Float64 >= 2.5:
		return "Стабильное состояние с периодами напряжения. Рекомендуется внедрить регулярные практики релаксации и следить за режимом дня."
	
	case avgScore.Float64 >= 1.5:
		return "Требуется внимание к психологическому состоянию. Рекомендуется консультация специалиста и регулярный мониторинг самочувствия."
	
	default:
		return "Рекомендуется профессиональная консультация. Ваше состояние требует внимания специалиста."
	}
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