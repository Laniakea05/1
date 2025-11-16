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
		tests.Use(middleware.AuthRequired())
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
			// Статистика
			admin.GET("/stats", handlers.GetAdminStats)
			
			// Пользователи
			admin.GET("/users", handlers.GetAllUsers)
			admin.POST("/users/:id/block", handlers.BlockUser)
			
			// Тесты
			admin.GET("/tests", handlers.GetAllTests)
			admin.GET("/tests/:id/edit", handlers.GetTestForEdit)
			admin.POST("/tests", handlers.CreateTest)
			admin.PUT("/tests/:id", handlers.UpdateTest)
			admin.DELETE("/tests/:id", handlers.DeleteTest)
			
			// Результаты
			admin.GET("/results", handlers.GetAllResults)
		}

		// Health check
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":   "OK", 
				"message":  "Server is running with database",
				"database": "connected",
			})
		})

		// Debug routes
		api.GET("/debug/db-structure", handlers.CheckDBStructure)
	}

	// Frontend routes
	router.GET("/", handlers.IndexPage)
	router.GET("/login", handlers.LoginPage)
	router.GET("/register", handlers.RegisterPage)
	router.GET("/dashboard", handlers.DashboardPage)
	router.GET("/tests", handlers.TestsPage)
	router.GET("/test/:id", handlers.TestTakingPage)
	router.GET("/test-result", handlers.TestResultPage)
	router.GET("/admin", handlers.AdminPage)
	router.GET("/admin/test-edit", handlers.TestEditPage) // НОВЫЙ РОУТ

	log.Println("🚀 Server starting on http://localhost:8080")
	router.Run(":8080")
}