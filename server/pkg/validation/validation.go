package validation

type LoginRegister struct {
	Username string `json:"username" validate:"required,gte=2,lte=16"`
	Password string `json:"password" validate:"required,gte=8,lte=72"`
}

type CreateRoom struct {
	Name string `json:"name" validate:"required,gte=24,lte=2"`
}
