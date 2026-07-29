package security

import (
	"time"

	"pisowifi/internal/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gofiber/fiber/v2"
)

const tokenExpiry = 30 * time.Minute

// Claims is the JWT payload.
type Claims struct {
	Sub string `json:"sub"`
	jwt.RegisteredClaims
}

// CreateToken generates a signed HS256 JWT for the given username.
func CreateToken(username string) (string, error) {
	claims := Claims{
		Sub: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.SecretKey))
}

// VerifyToken validates a JWT string and returns the username if valid.
func VerifyToken(tokenStr string) (string, bool) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fiber.ErrUnauthorized
		}
		return []byte(config.SecretKey), nil
	})
	if err != nil || !token.Valid {
		return "", false
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return "", false
	}
	return claims.Sub, true
}

// AdminMiddleware is a Fiber middleware that protects admin routes.
// On failure it redirects to /login (matching Python's 302 behavior).
func AdminMiddleware(c *fiber.Ctx) error {
	token := c.Cookies("admin_token")
	if token == "" {
		return c.Redirect("/login", fiber.StatusFound)
	}
	username, ok := VerifyToken(token)
	if !ok || username != config.AdminUsername {
		return c.Redirect("/login", fiber.StatusFound)
	}
	c.Locals("admin_user", username)
	return c.Next()
}
