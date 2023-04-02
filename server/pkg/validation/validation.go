package validation

type LoginRegister struct {
	Username string `json:"username" validate:"required,gte=2,lte=16"`
	Password string `json:"password" validate:"required,gte=8,lte=72"`
}

type CreateUpdateRoom struct {
	Name    string `json:"name" validate:"required,gte=2,lte=16"`
	Private bool   `json:"private"`
}

type GetUserByName struct {
	Username string `json:"username" validate:"required,gte=2,lte=16"`
}

type Bio struct {
	Content string `json:"content" validate:"lte=300"`
}

type CreateUpdateChannel struct {
	Name string `json:"name" validate:"required,lte=16,gte=2"`
	Main bool   `json:"main"`
}
