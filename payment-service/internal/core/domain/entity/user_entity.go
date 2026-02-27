package entity

type UserHttpClientResponse struct {
	Message string           `json:"message"`
	Data    UserHttpResponse `json:"data"`
}

type UserHttpResponse struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Address  string `json:"address"`
	Lat      string `json:"lat"`
	Lng      string `json:"lng"`
	RoleName string `json:"role"`
	Photo    string `json:"photo"`
}
