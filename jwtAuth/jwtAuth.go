package jwtauth

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gofiber/fiber/v2"
)

type Config struct {
	Next         func(c *fiber.Ctx) bool
	Unauthorized fiber.Handler
	Decode       func(c *fiber.Ctx) (*jwt.MapClaims, error)
	Secret       string
	Expiration   int64
}

var ConfigDefault = Config{
	Next:         nil,
	Unauthorized: nil,
	Decode:       nil,
	Secret:       os.Getenv("JWT_SECRET"),
	Expiration:   60,
}

func configDefault(config ...Config) Config {
	if len(config) < 1 {
		return ConfigDefault
	}

	cfg := config[0]

	if cfg.Next == nil {
		cfg.Next = ConfigDefault.Next
	}

	if cfg.Secret == "" {
		cfg.Secret = ConfigDefault.Secret
	}

	if cfg.Expiration == 0 {
		cfg.Expiration = ConfigDefault.Expiration
	}

	if cfg.Decode == nil {
		cfg.Decode = func(c *fiber.Ctx) (*jwt.MapClaims, error) {
			authHeader := c.Get("Authorization")

			if authHeader == "" {
				return nil, errors.New("missing auth header")
			}

			token, err := jwt.Parse(authHeader[7:], func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method %v", token.Header["alg"])
				}
				return []byte(cfg.Secret), nil
			})
			if err != nil {
				return nil, errors.New("error parsing token")
			}

			claims, ok := token.Claims.(jwt.MapClaims)

			if !(ok && token.Valid) {
				return nil, errors.New("invalid token")
			}

			if expiresAt, ok := claims["exp"]; ok && int64(expiresAt.(float64)) < time.Now().UTC().Unix() {
				return nil, errors.New("jwt expired")
			}

			return &claims, nil
		}
	}

	if cfg.Unauthorized == nil {
		cfg.Unauthorized = func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
	}

	return cfg
}

func Encode(claims *jwt.MapClaims, expireAfter int64) (string, error) {

	if expireAfter == 0 {
		expireAfter = ConfigDefault.Expiration
	}

	(*claims)["exp"] = time.Now().UTC().Unix() + expireAfter
	(*claims)["iss"] = os.Getenv("JWT_ISS_NAME")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString([]byte(ConfigDefault.Secret))

	if err != nil {
		return "", errors.New("error creating a token")
	}

	return signedToken, nil
}

func New(config Config) fiber.Handler {
	cfg := configDefault(config)

	return func(c *fiber.Ctx) error {
		if cfg.Next != nil && cfg.Next(c) {
			return c.Next()
		}

		claims, err := cfg.Decode(c)

		if err == nil {
			c.Locals("jwtClaims", *claims)
			return c.Next()
		}

		return cfg.Unauthorized(c)
	}
}
