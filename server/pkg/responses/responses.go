package responses

type User struct {
	ID       string `json:"ID"`
	Username string `json:"username"`
	// "ADMIN" | "USER"
	Role string `json:"role"`
}

type Room struct {
	ID        string `json:"ID"`
	Name      string `json:"name"`
	AuthorID  string `json:"author_id"`
	Private   bool   `json:"private"`
	CreatedAt string `json:"created_at"`
}

type RoomMessage struct {
	ID        string `json:"ID"`
	Content   string `json:"content"`
	AuthorID  string `json:"author_id"`
	CreatedAt string `json:"created_at"`
}
