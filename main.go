package main

import (
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/joho/godotenv"
	"gitlab.com/niles87-microservices/main-auth-server/controller"
	"gitlab.com/niles87-microservices/main-auth-server/mydb"
)

func main() {
	// Load dotenv file
	envErr := godotenv.Load()
	if envErr != nil {
		log.Print("ENV failed to load")
	}

	db, err := mydb.Connect()
	if err != nil {
		log.Fatal(err)
	}

	hdl := controller.NewDBHandler(db)

	// Set port (for heroku later)
	PORT := os.Getenv("PORT")

	// Initialize app
	app := fiber.New(fiber.Config{})

	// Add middleware with .Use
	app.Use(logger.New())
	app.Use(requestid.New())
	app.Use(limiter.New(limiter.Config{
		Max:        10,
		Expiration: 1 * time.Minute,
	}))

	// Group related endpoints together
	userApp := app.Group("/user")
	userApp.Get("", hdl.GetUsers)
	userApp.Post("", hdl.CreateUser)
	userApp.Put("", hdl.UpdateUser)
	userApp.Get("/:id", hdl.GetUserById)
	userApp.Delete("/:id", hdl.DeleteUser)
	userApp.Post("/login", hdl.Login)

	log.Fatal(app.Listen(":" + PORT))
}
