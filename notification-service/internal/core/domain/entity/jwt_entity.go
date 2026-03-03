package entity

import "time"

type JwtUserData struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	LoggedIn  bool      `json:"logged_in"`
	Token     string    `json:"token"`
	RoleName  string    `json:"role_name"`
	CreatedAt time.Time `json:"created_at"`
}
