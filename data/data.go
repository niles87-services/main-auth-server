package data

type Message struct {
	Msg string
}

type User struct {
	Id       int64
	Name     string
	Email    string
	Password string
}

type UserDto struct {
	Id    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type UpdateUserDto struct {
	Name             string
	Email            string
	NewPassword      string
	ExistingPassword string
}
