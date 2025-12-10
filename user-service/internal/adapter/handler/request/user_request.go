package request

type SignInRequest struct {
	Email    string `json:"email" validate:"email,required"`
	Password string `json:"password" validate:"min=6,required"`
}

type SignUpRequest struct {
	Name                 string `json:"name" validate:"required"`
	Email                string `json:"email" validate:"email,required"`
	Password             string `json:"password" validate:"min=6,required"`
	PasswordConfirmation string `json:"password_confirmation" validate:"eqfield=Password"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"email,required"`
}

type UpdatePasswordRequest struct {
	Token           string `json:"token" validate:"required"`
	CurrentPassword string `json:"password" validate:"omitempty"`
	NewPassword     string `json:"password_new" validate:"min=6,required"`
	ConfirmPassword string `json:"password_confirmation" validate:"eqfield=NewPassword"`
}

type UpdateDataUserRequest struct {
	Name    string `json:"name" validate:"required"`
	Email   string `json:"email" validate:"required,email"`
	Phone   string `json:"phone" validate:"required,min=10,max=20"`
	Address string `json:"address" validate:"required"`
	Lat     string `json:"lat" validate:"required"`
	Lng     string `json:"lng" validate:"required"`
	Photo   string `json:"photo" validate:"required"`
}
