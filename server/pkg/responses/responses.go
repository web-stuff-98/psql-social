package responses

type User struct {
	Username string `json:"username"`
	// "ADMIN" | "OWNER" | "USER"
	Role string `json:"role"`
}

type UserWithToken struct {
	Username string `json:"username"`
	// "ADMIN" | "OWNER" | "USER"
	Role  string `json:"role"`
	Token string `json:"token"`
}
