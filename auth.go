package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

const (
	roleCreator   = "creator"
	roleModerator = "moderator"
)

type authUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type jwtClaims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type loginResponse struct {
	Token string `json:"token"`
	Role  string `json:"role"`
	User  struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
	} `json:"user"`
}

// --------------- Redis (blacklist) ---------------

var (
	redisOnce   sync.Once
	redisClient *redis.Client
	redisErr    error
)

func getRedisClient() (*redis.Client, error) {
	redisOnce.Do(func() {
		addr := getEnv("REDIS_ADDR", "localhost:6380")
		password := getEnv("REDIS_PASSWORD", "")
		redisClient = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       0,
		})
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		redisErr = redisClient.Ping(ctx).Err()
	})
	return redisClient, redisErr
}

func addToBlacklist(ctx context.Context, tokenStr string, expiration time.Duration) error {
	rc, err := getRedisClient()
	if err != nil {
		return err
	}
	return rc.Set(ctx, "bl:"+tokenStr, "1", expiration).Err()
}

func isBlacklisted(ctx context.Context, tokenStr string) bool {
	rc, err := getRedisClient()
	if err != nil {
		return false
	}
	val, err := rc.Exists(ctx, "bl:"+tokenStr).Result()
	return err == nil && val > 0
}

// --------------- JWT ---------------

func jwtSecret() string {
	return getEnv("JWT_SECRET", "lab4-very-secret-key-change-me")
}

func generateJWT(u authUser) (string, error) {
	now := time.Now()
	claims := jwtClaims{
		UserID:   u.ID,
		Username: u.Username,
		Role:     u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret()))
}

func parseJWT(tokenStr string) (*jwtClaims, error) {
	claims := &jwtClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(jwtSecret()), nil
	})
	if err != nil || token == nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// --------------- Роли ---------------

func roleForUser(id int, username string) string {
	if id == 2 || strings.EqualFold(username, "petrova") {
		return roleModerator
	}
	return roleCreator
}

// --------------- Схема БД ---------------

func ensureAuthSchema() {
	if _, err := db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255)`); err != nil {
		fmt.Printf("auth schema warning (password_hash): %v\n", err)
		return
	}
	seedUsers := []string{"ivanov", "petrova", "sidorov"}
	for _, username := range seedUsers {
		var current sql.NullString
		if err := db.QueryRow(`SELECT password_hash FROM users WHERE username = $1`, username).Scan(&current); err != nil {
			continue
		}
		if current.Valid && strings.TrimSpace(current.String) != "" {
			continue
		}
		hash, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
		if err != nil {
			continue
		}
		_, _ = db.Exec(`UPDATE users SET password_hash = $1 WHERE username = $2 AND (password_hash IS NULL OR password_hash = '')`, string(hash), username)
	}
}

// --------------- Auth middleware ---------------

func extractBearerToken(r *http.Request) string {
	authz := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(authz, "Bearer ") {
		return strings.TrimSpace(authz[len("Bearer "):])
	}
	return ""
}

func authUserFromRequest(r *http.Request) (*authUser, string, error) {
	tokenStr := extractBearerToken(r)
	if tokenStr == "" {
		return nil, "", errors.New("missing Authorization header")
	}
	if isBlacklisted(r.Context(), tokenStr) {
		return nil, "", errors.New("token revoked")
	}
	claims, err := parseJWT(tokenStr)
	if err != nil {
		return nil, "", err
	}
	return &authUser{
		ID:       claims.UserID,
		Username: claims.Username,
		Role:     claims.Role,
	}, tokenStr, nil
}

func requireAuth(w http.ResponseWriter, r *http.Request) (*authUser, bool) {
	u, _, err := authUserFromRequest(r)
	if err != nil || u == nil {
		writeError(w, http.StatusUnauthorized, "Требуется авторизация (заголовок Authorization: Bearer <token>)")
		return nil, false
	}
	return u, true
}

func canAccessPosition(u *authUser, pos *ISSPosition) bool {
	if u == nil || pos == nil {
		return false
	}
	if u.Role == roleModerator {
		return true
	}
	return u.ID == pos.CreatorID
}

// --------------- Swagger ---------------

func swaggerUIHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/swagger" && r.URL.Path != "/swagger/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!doctype html>
<html lang="ru">
<head>
  <meta charset="utf-8">
  <title>ISS Visibility API - Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: '/openapi.json',
      dom_id: '#swagger-ui',
      deepLinking: true,
      presets: [SwaggerUIBundle.presets.apis],
      requestInterceptor: function(req) {
        var t = localStorage.getItem('iss_jwt');
        if (t) req.headers['Authorization'] = 'Bearer ' + t;
        return req;
      }
    });
  </script>
</body>
</html>`)
}

func openAPISpecHandler(w http.ResponseWriter, _ *http.Request) {
	data, err := os.ReadFile("openapi/openapi.json")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "OpenAPI spec not found: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func writeAuthLoginResponse(w http.ResponseWriter, u authUser, token string) {
	var resp loginResponse
	resp.Token = token
	resp.Role = u.Role
	resp.User.ID = u.ID
	resp.User.Username = u.Username
	writeJSON(w, http.StatusOK, resp)
}
