package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func getDBConfig() (host string, port string, user string, password string, dbname string) {
	host = getEnv("DB_HOST", "postgres")
	port = getEnv("DB_PORT", "5432")
	user = getEnv("DB_USER", "postgres")
	password = getEnv("DB_PASSWORD", "postgres")
	dbname = getEnv("DB_NAME", "psycho_test_system")
	return
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func InitDB() (*sql.DB, error) {
	host, port, user, password, dbname := getDBConfig()
	
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	log.Printf("Connecting to database: %s@%s:%s", user, host, port)
	
	var err error
	DB, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Настраиваем пул соединений
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(25)
	DB.SetConnMaxLifetime(5 * time.Minute)

	// Пробуем подключиться с повторными попытками (ждём пока PostgreSQL запустится)
	for i := 0; i < 10; i++ {
		err = DB.Ping()
		if err == nil {
			break
		}
		log.Printf("Attempt %d: Failed to connect to database: %v", i+1, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database after retries: %v", err)
	}

	log.Println("✅ Successfully connected to PostgreSQL database!")
	return DB, nil
}