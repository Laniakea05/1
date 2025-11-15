package main

import (
	"log"
	"psycho-test-system/database"
	"psycho-test-system/handlers"
	"psycho-test-system/middleware"

	"github.com/gin-gonic/gin"
)

func main() {
	// Инициализация базы данных
	db, err := database.InitDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	log.Println("✅ Database connected successfully!")

	// Создаём тестовых пользователей при первом запуске
	handlers.CreateTestUsers()

	router := gin.Default()

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Статические файлы
	router.Static("/static", "./frontend")
	router.LoadHTMLGlob("./frontend/*.html")

	// API Routes
	api := router.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/login", handlers.Login)
			auth.POST("/register", handlers.Register)
		}

		tests := api.Group("/tests")
		tests.Use(middleware.AuthRequired()) // Добавляем аутентификацию для тестов
		{
			tests.GET("", handlers.GetTests)
			tests.GET("/:id", handlers.GetTest)
			tests.POST("/:id/submit", handlers.SubmitTest)
		}

		user := api.Group("/user")
		user.Use(middleware.AuthRequired())
		{
			user.GET("/profile", handlers.GetUserProfile)
			user.GET("/stats", handlers.GetUserStats)
			user.PUT("/profile", handlers.UpdateUserProfile)
		}

		admin := api.Group("/admin")
		admin.Use(middleware.AuthRequired())
		admin.Use(middleware.AdminRequired())
		{
			admin.POST("/tests", handlers.CreateTest)
			admin.PUT("/tests/:id", handlers.UpdateTest)
			admin.DELETE("/tests/:id", handlers.DeleteTest)
		}

		// Health check
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":   "OK", 
				"message":  "Server is running with database",
				"database": "connected",
			})
		})
	}

	// Frontend routes
	router.GET("/", handlers.IndexPage)
	router.GET("/login", handlers.LoginPage)
	router.GET("/register", handlers.RegisterPage)
	router.GET("/dashboard", handlers.DashboardPage)
	router.GET("/tests", handlers.TestsPage)
	router.GET("/test/:id", handlers.TestTakingPage)
	router.GET("/admin", handlers.AdminPage)

	log.Println("🚀 Server starting on http://localhost:8080")
	router.Run(":8080")
}