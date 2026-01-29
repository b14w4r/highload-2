package models

import (
	"errors"
	"strings"
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (u User) Validate() error {
	if strings.TrimSpace(u.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(u.Email) == "" {
		return errors.New("email is required")
	}
	// Упрощённая валидация, чтобы не тянуть лишние зависимости
	if !strings.Contains(u.Email, "@") {
		return errors.New("email must contain '@'")
	}
	return nil
}

