package request

type SignInRequest struct {
	Email    string `json:"email" validate:"email,required,max=255"`
	Password string `json:"password" validate:"min=6,required,max=72"`
}

type SignUpRequest struct {
	Name                 string `json:"name" validate:"required,max=255"`
	Email                string `json:"email" validate:"email,required,max=255"`
	Password             string `json:"password" validate:"min=6,required,max=72"`
	PasswordConfirmation string `json:"password_confirmation" validate:"eqfield=Password"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"email,required,max=255"`
}

type UpdatePasswordRequest struct {
	Token           string `json:"token" validate:"required,max=255"`
	CurrentPassword string `json:"password" validate:"omitempty"`
	NewPassword     string `json:"password_new" validate:"min=6,required,max=72"`
	ConfirmPassword string `json:"password_confirmation" validate:"eqfield=NewPassword"`
}

type UpdateProfilePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required,min=6,max=72"`
	NewPassword     string `json:"new_password" validate:"required,min=6,max=72"`
	ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=NewPassword"`
}

type UpdateDataUserRequest struct {
	Name    string `json:"name" validate:"required,max=255"`
	Email   string `json:"email" validate:"required,email,max=255"`
	Phone   string `json:"phone" validate:"required,min=10,max=17"`
	Address string `json:"address" validate:"required,max=500"`
	Lat     string `json:"lat" validate:"required,max=50"`
	Lng     string `json:"lng" validate:"required,max=50"`
	Photo   string `json:"photo" validate:"required,max=255"`
}
