package controller

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"gitlab.com/niles87-microservices/main-auth-server/data"
	"gitlab.com/niles87-microservices/main-auth-server/helpers"
	jwtauth "gitlab.com/niles87-microservices/main-auth-server/jwtAuth"
)

type DBHandler struct {
	db *sql.DB
}

// NewDBHandler accepts a pointer to a sql database connection.
// Returns a pointer to a DBHandler struct.
func NewDBHandler(db *sql.DB) *DBHandler {
	return &DBHandler{
		db: db,
	}
}

func (db *DBHandler) GetUsers(c *fiber.Ctx) error {
	users, err := queryAllUsers(db)
	if err != nil {
		fmt.Println(err)
		c.Status(fiber.StatusInternalServerError).JSON(data.Message{Msg: "something failed"})
		return err
	}

	return c.Status(fiber.StatusOK).JSON(users)
}

func (db *DBHandler) GetUserById(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		c.Status(fiber.StatusNotFound).JSON(data.Message{Msg: "Missing params"})
		return err
	}
	user, err := queryUserByID(db, int64(id))
	if err != nil {
		c.Status(fiber.StatusNotFound).JSON(data.Message{Msg: "User not found"})
		return err
	}

	return c.Status(fiber.StatusOK).JSON(user)
}

func (db *DBHandler) CreateUser(c *fiber.Ctx) error {

	body := new(data.User)

	err := c.BodyParser(body)

	if err != nil {
		c.Status(fiber.StatusBadRequest).JSON(data.Message{Msg: err.Error()})
		return err
	}

	hashedPassword, err := helpers.HashPassword(body.Password)
	if err != nil {
		fmt.Println(err)
		c.Status(fiber.StatusInternalServerError).JSON(data.Message{Msg: "Password failure"})
		return err
	}

	user := data.User{
		Name:     body.Name,
		Email:    body.Email,
		Password: hashedPassword,
	}

	id, err := addUser(db, user)

	if err != nil {
		fmt.Println(err)
		c.Status(fiber.StatusInternalServerError).JSON(data.Message{Msg: "something failed"})
		return err
	}

	user.Id = id

	userDto := data.UserDto{
		Id:    user.Id,
		Name:  user.Name,
		Email: user.Email,
	}

	return c.Status(fiber.StatusCreated).JSON(userDto)
}

func (db *DBHandler) UpdateUser(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		c.Status(fiber.StatusNotFound).JSON(data.Message{Msg: "Missing params"})
		return err
	}
	body := new(data.UpdateUserDto)
	err = c.BodyParser(body)
	if err != nil {
		c.Status(fiber.StatusBadRequest).JSON(data.Message{Msg: err.Error()})
		return err
	}

	updateUserDto := *body
	hashedPassword, err := helpers.HashPassword(updateUserDto.NewPassword)
	if err != nil {
		fmt.Println(err)
		c.Status(fiber.StatusInternalServerError).JSON(data.Message{Msg: "Password failure"})
		return err
	}

	validPassword := helpers.CheckPassword(updateUserDto.ExistingPassword, hashedPassword)

	if !validPassword {
		c.Status(fiber.StatusInternalServerError).JSON(data.Message{Msg: "Internal service error"})
		return errors.New("internal server error")
	}

	user := data.User{
		Id:       int64(id),
		Name:     updateUserDto.Name,
		Email:    updateUserDto.Email,
		Password: hashedPassword,
	}

	rowAffected, err := updateUserByID(db, int64(id), user)
	if err != nil {
		fmt.Println(err)
		c.Status(fiber.StatusInternalServerError).JSON(data.Message{Msg: "something failed"})
		return err
	}

	if rowAffected == 1 {
		userDto := data.UserDto{
			Id:    user.Id,
			Name:  user.Name,
			Email: user.Email,
		}
		return c.Status(fiber.StatusAccepted).JSON(userDto)
	}
	return c.Status(fiber.StatusNotFound).JSON(data.Message{Msg: "User Not Found"})
}

func (db *DBHandler) DeleteUser(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		c.Status(fiber.StatusNotFound).JSON(data.Message{Msg: "Missing params"})
		return err
	}

	rowsRemoved, err := deleteUserByID(db, int64(id))
	if err != nil {
		c.Status(fiber.StatusNotFound).JSON(data.Message{Msg: "User not found"})
		return err
	}

	return c.Status(fiber.StatusAccepted).JSON(data.Message{Msg: fmt.Sprintf("Success %d record removed", rowsRemoved)})
}

func (db *DBHandler) Login(c *fiber.Ctx) error {
	body := new(data.User)

	err := c.BodyParser(body)
	if err != nil {
		c.Status(fiber.StatusBadRequest).JSON(data.Message{Msg: err.Error()})
		return err
	}

	user, err := queryUserByEmail(db, body.Email)
	if err != nil {
		c.Status(fiber.StatusBadRequest).JSON(data.Message{Msg: "User not found"})
		return err
	}

	match := helpers.CheckPassword(body.Password, user.Password)

	if match {
		// Need to add token login
		userDto := data.UserDto{
			Id:    user.Id,
			Email: user.Email,
			Name:  user.Name,
		}

		token, err := jwtauth.Encode(&jwt.MapClaims{
			"id":   userDto.Id,
			"name": user.Name,
		}, 2000)

		if err != nil {
			return c.SendStatus(500)
		}

		c.Set("Authorization", "Bearer "+token)

		return c.Status(fiber.StatusOK).JSON(userDto)
	} else {
		return c.Status(fiber.StatusBadRequest).JSON(data.Message{Msg: "Record not found"})
	}
}

func queryAllUsers(hdl *DBHandler) ([]data.UserDto, error) {
	var users []data.UserDto
	rows, err := hdl.db.Query("SELECT * FROM user")
	if err != nil {
		return nil, fmt.Errorf("queryAllUsers: %v", err)
	}

	defer rows.Close()

	for rows.Next() {
		var user data.UserDto
		if err := rows.Scan(&user.Id, &user.Name, &user.Email); err != nil {
			return nil, fmt.Errorf("queryAllUsers: %v", err)
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("queryAllUsers: %v", err)
	}

	return users, nil
}

func addUser(hdl *DBHandler, user data.User) (int64, error) {
	res, err := hdl.db.Exec("INSERT INTO user (name, email, password) VALUES (?,?,?)", user.Name, user.Email, user.Password)

	if err != nil {
		return 0, fmt.Errorf("addUser %v", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("addUser %v", err)
	}

	return id, nil
}

func queryUserByID(hdl *DBHandler, id int64) (data.UserDto, error) {
	var user data.UserDto

	row := hdl.db.QueryRow("SELECT * FROM user WHERE id=?", id)

	if err := row.Scan(&user.Id, &user.Name, &user.Email); err != nil {
		if err == sql.ErrNoRows {
			return user, fmt.Errorf("queryUserById no record with id: %d ", id)
		}
		return user, fmt.Errorf("queryUserById %v", err)
	}

	return user, nil
}

func deleteUserByID(hdl *DBHandler, id int64) (int64, error) {
	stmt, err := hdl.db.Prepare("DELETE FROM user WHERE id=?")
	if err != nil {
		return 0, fmt.Errorf("failed to prepare statement: %v", err)
	}

	res, err := stmt.Exec(id)
	if err != nil {
		return 0, fmt.Errorf("deleteUserByID: %v", err)
	}

	rowsRemoved, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("deleteUserById: %v", err)
	}

	return rowsRemoved, nil
}

func updateUserByID(hdl *DBHandler, id int64, user data.User) (int64, error) {
	stmt, err := hdl.db.Prepare("UPDATE user SET name=?, email=?, password=? WHERE id=?")
	if err != nil {
		return 0, fmt.Errorf("failed to prepare statement: %v", err)
	}

	res, err := stmt.Exec(user.Name, user.Email, user.Password, id)
	if err != nil {
		return 0, fmt.Errorf("updateUserById: %v", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("updateUserById: %v", err)
	}

	return rowsAffected, nil
}

func queryUserByEmail(hdl *DBHandler, email string) (data.User, error) {
	var user data.User

	row := hdl.db.QueryRow("SELECT * FROM user WHERE email=?", email)

	if err := row.Scan(&user.Id, &user.Name, &user.Email, &user.Password); err != nil {
		if err == sql.ErrNoRows {
			return user, fmt.Errorf("queryUserByEmail no record with email: %s ", email)
		}
		return user, fmt.Errorf("queryUserByEmail %v", err)
	}

	return user, nil
}
