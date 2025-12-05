package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"psycho-test-system/database"
	"psycho-test-system/models"
	"psycho-test-system/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// Функция для создания тестовых пользователей при первом запуске
func CreateTestUsers() {
	// Проверяем есть ли уже пользователи с правильными паролями
	var count int
	err := database.DB.QueryRow("SELECT COUNT(*) FROM users WHERE password_hash != 'temp_password'").Scan(&count)
	if err != nil {
		fmt.Printf("Ошибка проверки пользователей: %v\n", err)
		return
	}
	
	if count > 0 {
		fmt.Println("✅ Пользователи уже существуют с правильными паролями")
		return
	}

	fmt.Println("🔄 Создаем тестовых пользователей с правильными паролями...")

	// Создаём тестовых пользователей
	users := []struct {
		email      string
		password   string
		lastName   string
		firstName  string
		patronymic string
		role       string
	}{
		{"admin@psycho.test", "admin123", "Администратор", "Системы", "", "admin"},
		{"user@test.ru", "user123", "Пользователь", "Тестовый", "Тестович", "user"},
	}

	for _, u := range users {
		hashedPassword, err := utils.HashPassword(u.password)
		if err != nil {
			fmt.Printf("❌ Ошибка хеширования пароля для %s: %v\n", u.email, err)
			continue
		}
		
		fmt.Printf("Обновляем пользователя: %s (%s)\n", u.email, u.role)
		
		_, err = database.DB.Exec(
			"UPDATE users SET password_hash = $1 WHERE email = $2",
			hashedPassword, u.email,
		)
		if err != nil {
			fmt.Printf("❌ Ошибка обновления пользователя %s: %v\n", u.email, err)
			continue
		}
		
		fmt.Printf("✅ Пользователь %s обновлен успешно!\n", u.email)
	}
	
	fmt.Println("✅ Все тестовые пользователи обновлены с правильными паролями!")
}

// Функция проверки на русские буквы
func containsRussianLetters(text string) bool {
	re := regexp.MustCompile(`[а-яА-ЯёЁ]`)
	return re.MatchString(text)
}

// Функция проверки формата email
func isValidEmailFormat(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}

// Проверка валидности имени (только буквы, пробелы и дефисы)
func isValidName(name string) bool {
	re := regexp.MustCompile(`^[a-zA-Zа-яА-ЯёЁ\s\-]+$`)
	return re.MatchString(name)
}

// CheckEmail проверяет доступность email
func CheckEmail(c *gin.Context) {
	var checkReq struct {
		Email string `json:"email" binding:"required"`
	}

	if err := c.ShouldBindJSON(&checkReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат email"})
		return
	}

	// Проверяем на русские буквы
	if containsRussianLetters(checkReq.Email) {
		c.JSON(http.StatusBadRequest, gin.H{
			"available": false,
			"error": "Email не должен содержать русские буквы. Используйте только английские буквы, цифры и символы @._-",
			"email": checkReq.Email,
		})
		return
	}

	// Проверяем общий формат email
	if !isValidEmailFormat(checkReq.Email) {
		c.JSON(http.StatusBadRequest, gin.H{
			"available": false,
			"error": "Неверный формат email. Пример: example@mail.ru",
			"email": checkReq.Email,
		})
		return
	}

	// Проверяем существует ли пользователь с таким email
	var exists bool
	err := database.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)",
		checkReq.Email,
	).Scan(&exists)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка проверки email"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"available": !exists,
		"email":     checkReq.Email,
	})
}

func Login(c *gin.Context) {
	var loginReq models.LoginRequest
	if err := c.ShouldBindJSON(&loginReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный запрос"})
		return
	}

	// Ищем пользователя в БД
	var user models.User
	var isBlocked bool
	err := database.DB.QueryRow(
		"SELECT id, email, password_hash, last_name, first_name, patronymic, role, is_blocked FROM users WHERE email = $1",
		loginReq.Email,
	).Scan(&user.ID, &user.Email, &user.Password, &user.LastName, &user.FirstName, &user.Patronymic, &user.Role, &isBlocked)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный email или пароль"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка базы данных: " + err.Error()})
		return
	}

	// Проверяем заблокирован ли пользователь
	if isBlocked {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Пользователь заблокирован"})
		return
	}

	// ПРАВИЛЬНАЯ проверка пароля через bcrypt
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginReq.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный email или пароль"})
		return
	}

	// Генерируем JWT токен
	token, err := utils.GenerateJWT(user.ID, user.Email, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка генерации токена"})
		return
	}

	// Успешный вход
	c.JSON(http.StatusOK, gin.H{
		"message": "✅ Вход выполнен успешно!",
		"token":   token,
		"user": gin.H{
			"id":         user.ID,
			"email":      user.Email,
			"last_name":  user.LastName,
			"first_name": user.FirstName,
			"patronymic": user.Patronymic,
			"full_name":  user.LastName + " " + user.FirstName + " " + user.Patronymic,
			"role":       user.Role,
		},
	})
}

func Register(c *gin.Context) {
	var registerReq models.RegisterRequest
	if err := c.ShouldBindJSON(&registerReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный запрос"})
		return
	}

	// Проверяем email на русские буквы
	if containsRussianLetters(registerReq.Email) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Email не должен содержать русские буквы. Используйте только английские буквы, цифры и символы @._-",
		})
		return
	}

	// Проверяем формат email
	if !isValidEmailFormat(registerReq.Email) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Неверный формат email. Пример: example@mail.ru",
		})
		return
	}

	// Проверяем имена на валидность
	if !isValidName(registerReq.LastName) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Фамилия должна содержать только буквы, пробелы и дефисы",
		})
		return
	}

	if !isValidName(registerReq.FirstName) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Имя должно содержать только буквы, пробелы и дефисы",
		})
		return
	}

	if registerReq.Patronymic != "" && !isValidName(registerReq.Patronymic) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Отчество должно содержать только буквы, пробелы и дефисы",
		})
		return
	}

	// Хешируем пароль ПРАВИЛЬНО
	hashedPassword, err := utils.HashPassword(registerReq.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка хеширования пароля"})
		return
	}

	// Создаем пользователя в БД
	var userID int
	err = database.DB.QueryRow(
		"INSERT INTO users (email, password_hash, last_name, first_name, patronymic, role, is_blocked) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id",
		registerReq.Email, hashedPassword, registerReq.LastName, registerReq.FirstName, registerReq.Patronymic, models.RoleUser, false,
	).Scan(&userID)

	if err != nil {
		// Проверяем, является ли ошибка нарушением уникальности email
		if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "duplicate key") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Пользователь с таким email уже существует"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания пользователя: " + err.Error()})
		}
		return
	}

	// Генерируем JWT токен
	token, err := utils.GenerateJWT(userID, registerReq.Email, models.RoleUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка генерации токена"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "✅ Пользователь зарегистрирован!",
		"token":   token,
		"user": gin.H{
			"id":         userID,
			"email":      registerReq.Email,
			"last_name":  registerReq.LastName,
			"first_name": registerReq.FirstName,
			"patronymic": registerReq.Patronymic,
			"full_name":  registerReq.LastName + " " + registerReq.FirstName + " " + registerReq.Patronymic,
			"role":       models.RoleUser,
		},
	})
}