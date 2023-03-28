package responses

type User struct {
	Username string `json:"username"`
	// "ADMIN" | "OWNER" | "USER"
	Role string `json:"role"`
}
