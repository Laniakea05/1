package handlers

import (
	"net/http"
	"strconv"
	"psycho-test-system/database"
	"psycho-test-system/models"

	"github.com/gin-gonic/gin"
)

func CreateTest(c *gin.Context) {
	var test models.PsychologicalTest
	if err := c.ShouldBindJSON(&test); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}

	userID, _ := c.Get("userID")
	
	var testID int
	err := database.DB.QueryRow(`
		INSERT INTO psychological_tests (title, description, instructions, estimated_time, created_by)
		VALUES ($1, $2, $3, $4, $5) RETURNING id
	`, test.Title, test.Description, test.Instructions, test.EstimatedTime, userID).Scan(&testID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания теста"})
		return
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