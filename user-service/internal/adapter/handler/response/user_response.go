package response

type SignInResponse struct {
	AccessToken string `json:"access_token"`
	Role        string `json:"role"`
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Address     string `json:"address"`
	Lat         string `json:"lat"`
	Lng         string `json:"lng"`
}

type ProfileResponse struct {
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

type CustomerResponse struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Address  string `json:"address"`
	Lat      string `json:"lat"`
	Lng      string `json:"lng"`
	Photo    string `json:"photo"`
	RoleID   int64  `json:"role_id"`
	RoleName string `json:"role,omitempty"`
}
