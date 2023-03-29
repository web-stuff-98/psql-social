package responses

type User struct {
	ID       string `json:"ID"`
	Username string `json:"username"`
	// "ADMIN" | "USER"
	Role string `json:"role"`
}

type UserWithToken struct {
	ID       string `json:"ID"`
	Username string `json:"username"`
	// "ADMIN" | "USER"
	Role  string `json:"role"`
	Token string `json:"token"`
}
