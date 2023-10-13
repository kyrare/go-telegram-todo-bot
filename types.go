package main

import (
	"time"
)

type User struct {
	ID         int
	TelegramId int
	FirstName  string
	UserName   string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Todo struct {
	ID        int
	UserId    int
	User      User
	Text      string
	Checked   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
