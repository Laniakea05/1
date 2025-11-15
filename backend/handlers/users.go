package handlers

import (
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
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения статистики"})
        return
    }

    // Средний балл
    err = database.DB.QueryRow(`
        SELECT COALESCE(AVG(score), 0) FROM test_results 
        WHERE user_id = $1
    `, userID).Scan(&stats.AverageScore)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения среднего балла"})
        return
    }

    // Дата последнего теста - исправленный запрос
    err = database.DB.QueryRow(`
        SELECT COALESCE(TO_CHAR(completed_at, 'DD.MM.YYYY'), '-') 
        FROM test_results 
        WHERE user_id = $1 
        ORDER BY completed_at DESC 
        LIMIT 1
    `, userID).Scan(&stats.LastTestDate)
    if err != nil {
        // Если ошибка - просто ставим прочерк
        stats.LastTestDate = "-"
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