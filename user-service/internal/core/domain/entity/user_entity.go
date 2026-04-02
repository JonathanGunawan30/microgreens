package entity

type UserEntity struct {
	ID         int64
	Name       string
	Email      string
	Password   string
	RoleName   string
	RoleID     int64
	Address    string
	Lat        string
	Lng        string
	Phone      string
	Photo      string
	IsVerified bool
	Token      string
}

type QueryStringCustomer struct {
	Search    string
	Page      int64
	Limit     int64
	OrderBy   string
	OrderType string
}

type UserEvent struct {
	UserID  int64  `json:"user_id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	Address string `json:"address"`
}
