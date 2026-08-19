package models

import "time"

// User backs the "users" table used by core.AuthService for both session
// and JWT authentication.
type User struct {
	ID        int64     `db:"id"`
	Name      string    `db:"name"`
	Email     string    `db:"email"`
	Password  string    `db:"password"`
	CreatedAt time.Time `db:"created_at"`
}

func (User) TableName() string { return "users" }
