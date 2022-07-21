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
	Id    int64
	Name  string
	Email string
}
