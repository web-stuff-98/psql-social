package validation

type LoginRegister struct {
	Username string `json:"username" validate:"required,gte=2,lte=16"`
	Password string `json:"password" validate:"required,gte=16,lte=72"`
}
