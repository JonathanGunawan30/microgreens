package request

type CustomerRequest struct {
	Name     string `json:"name"     validate:"required,max=255"`
	Email    string `json:"email"    validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=6,max=72"`
	Phone    string `json:"phone"    validate:"required,max=17"`
	Photo    string `json:"photo"    validate:"omitempty,max=255"`
	Address  string `json:"address"  validate:"required,max=500"`
	Lat      string `json:"lat"      validate:"required,max=50"`
	Lng      string `json:"lng"      validate:"required,max=50"`
}

type UpdateCustomerRequest struct {
	Name    string `json:"name" validate:"required,max=255"`
	Email   string `json:"email" validate:"required,email,max=255"`
	Phone   string `json:"phone" validate:"required,max=17"`
	Photo   string `json:"photo" validate:"omitempty,max=255"`
	Address string `json:"address" validate:"required,max=500"`
	Lat     string `json:"lat" validate:"required,max=50"`
	Lng     string `json:"lng" validate:"required,max=50"`
}
