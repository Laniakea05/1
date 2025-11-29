package models

import "time"

type User struct {
	ID         int       `json:"id"`
	Email      string    `json:"email"`
	Password   string    `json:"-"`
	LastName   string    `json:"last_name"`
	FirstName  string    `json:"first_name"`
	Patronymic string    `json:"patronymic"`
	Role       string    `json:"role"`
	CreatedAt  time.Time `json:"created_at"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=6"`
	LastName   string `json:"last_name" binding:"required"`
	FirstName  string `json:"first_name" binding:"required"`
	Patronymic string `json:"patronymic"`
}

const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)