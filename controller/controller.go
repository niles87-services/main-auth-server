package controller

import (
	"database/sql"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/niles87/my-go-services/data"
	"github.com/niles87/my-go-services/helpers"
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

	return c.Status(fiber.StatusCreated).JSON(user)
}

func (db *DBHandler) UpdateUser(c *fiber.Ctx) error {

	body := new(data.User)
	err := c.BodyParser(body)
	if err != nil {
		c.Status(fiber.StatusBadRequest).JSON(data.Message{Msg: err.Error()})
		return err
	}

	user := *body
	hashedPassword, err := helpers.HashPassword(user.Password)
	if err != nil {
		fmt.Println(err)
		c.Status(fiber.StatusInternalServerError).JSON(data.Message{Msg: "Password failure"})
		return err
	}
	user.Password = hashedPassword

	rowAffected, err := updateUserByID(db, user.Id, user)
	if err != nil {
		fmt.Println(err)
		c.Status(fiber.StatusInternalServerError).JSON(data.Message{Msg: "something failed"})
		return err
	}

	if rowAffected == 1 {
		return c.Status(fiber.StatusAccepted).JSON(user)
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
		return c.Status(fiber.StatusOK).JSON(user)
	} else {
		return c.Status(fiber.StatusBadRequest).JSON(data.Message{Msg: "Record not found"})
	}
}

func queryAllUsers(hdl *DBHandler) ([]data.User, error) {
	var users []data.User
	rows, err := hdl.db.Query("SELECT * FROM user")
	if err != nil {
		return nil, fmt.Errorf("queryAllUsers: %v", err)
	}

	defer rows.Close()

	for rows.Next() {
		var user data.User
		if err := rows.Scan(&user.Id, &user.Name, &user.Email, &user.Password); err != nil {
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

func queryUserByID(hdl *DBHandler, id int64) (data.User, error) {
	var user data.User

	row := hdl.db.QueryRow("SELECT * FROM user WHERE id=?", id)

	if err := row.Scan(&user.Id, &user.Name, &user.Email, &user.Password); err != nil {
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
