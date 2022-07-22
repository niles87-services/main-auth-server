package main

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/joho/godotenv"
	"gitlab.com/niles87-microservices/main-auth-server/controller"
	jwtauth "gitlab.com/niles87-microservices/main-auth-server/jwtAuth"
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
	app.Use(func(c *fiber.Ctx) error {
		c.Set("X-Frame-Options", "SAMEORIGIN")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("X-Content-Type-Options", "nosniff")
		return c.Next()
	})

	app.Use(jwtauth.New(jwtauth.Config{
		Next: func(c *fiber.Ctx) bool {
			return !strings.Contains(c.OriginalURL(), "/auth")
		},
	}))

	// Group related endpoints together
	userApp := app.Group("/user")
	userApp.Post("", hdl.CreateUser)
	userApp.Post("/login", hdl.Login)

	authUser := userApp.Group("/auth")
	authUser.Get("", hdl.GetUsers)
	authUser.Put("/:id", hdl.UpdateUser)
	authUser.Get("/:id", hdl.GetUserById)
	authUser.Delete("/:id", hdl.DeleteUser)

	log.Fatal(app.Listen(":" + PORT))
}
