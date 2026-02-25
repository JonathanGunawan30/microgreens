package request

type CustomerRequest struct {
	Name                 string `json:"name" validate:"required"`
	Email                string `json:"email" validate:"email,required"`
	Password             string `json:"password" validate:"min=6,required"`
	PasswordConfirmation string `json:"password_confirmation" validate:"eqfield=Password"`
	Phone                string `json:"phone" validate:"required"`
	Photo                string `json:"photo"`
	Address              string `json:"address" validate:"required"`
	Lat                  string `json:"lat" validate:"required"`
	Lng                  string `json:"lng" validate:"required"`
}

type UpdateCustomerRequest struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"omitempty,min=6"`
	Phone    string `json:"phone" validate:"required"`
	Photo    string `json:"photo"`
	Address  string `json:"address"`
	Lat      string `json:"lat"`
	Lng      string `json:"lng"`
}
