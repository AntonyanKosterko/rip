package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
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
	rb := func(required []string, props map[string]interface{}, example map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"required": true,
			"content": map[string]interface{}{
				"application/json": map[string]interface{}{
					"schema":  map[string]interface{}{"type": "object", "required": required, "properties": props},
					"example": example,
				},
			},
		}
	}
	jr := func(desc string, schema, example map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"description": desc,
			"content":     map[string]interface{}{"application/json": map[string]interface{}{"schema": schema, "example": example}},
		}
	}
	bearerSec := []map[string]interface{}{{"bearerAuth": []string{}}}

	spec := map[string]interface{}{
		"openapi": "3.0.3",
		"info": map[string]interface{}{
			"title":       "ISS Visibility API",
			"version":     "1.0.0-lab4",
			"description": "ЛР4: авторизация JWT + blacklist в Redis, роли creator/moderator.",
		},
		"servers": []map[string]string{{"url": "http://localhost:8080"}},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"paths": map[string]interface{}{
			"/api/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Логин: возвращает JWT-токен",
					"requestBody": rb(
						[]string{"username", "password"},
						map[string]interface{}{"username": map[string]interface{}{"type": "string"}, "password": map[string]interface{}{"type": "string"}},
						map[string]interface{}{"username": "ivanov", "password": "123456"},
					),
					"responses": map[string]interface{}{
						"200": jr("Успешная аутентификация",
							map[string]interface{}{"type": "object", "properties": map[string]interface{}{
								"token": map[string]interface{}{"type": "string"},
								"role":  map[string]interface{}{"type": "string"},
								"user":  map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "integer"}, "username": map[string]interface{}{"type": "string"}}},
							}},
							map[string]interface{}{"token": "eyJhbGciOi...", "role": "creator", "user": map[string]interface{}{"id": 1, "username": "ivanov"}},
						),
						"401": jr("Неверный логин или пароль", map[string]interface{}{"type": "object", "properties": map[string]interface{}{"error": map[string]interface{}{"type": "string"}}}, map[string]interface{}{"error": "Неверный логин или пароль"}),
					},
				},
			},
			"/api/auth/logout": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":  "Логаут: добавляет JWT в blacklist Redis",
					"security": bearerSec,
					"responses": map[string]interface{}{
						"200": jr("Токен отозван", map[string]interface{}{"type": "object", "properties": map[string]interface{}{"success": map[string]interface{}{"type": "boolean"}}}, map[string]interface{}{"success": true}),
					},
				},
			},
			"/api/users/register": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Регистрация пользователя",
					"requestBody": rb(
						[]string{"username", "full_name", "password"},
						map[string]interface{}{"username": map[string]interface{}{"type": "string"}, "full_name": map[string]interface{}{"type": "string"}, "email": map[string]interface{}{"type": "string"}, "password": map[string]interface{}{"type": "string"}},
						map[string]interface{}{"username": "newuser", "full_name": "Новый Пользователь", "email": "newuser@bmstu.ru", "password": "123456"},
					),
					"responses": map[string]interface{}{"201": map[string]interface{}{"description": "Пользователь зарегистрирован"}},
				},
			},
			"/api/services": map[string]interface{}{
				"get":  map[string]interface{}{"summary": "Список услуг (публично)", "responses": map[string]interface{}{"200": map[string]interface{}{"description": "Список услуг"}}},
				"post": map[string]interface{}{"summary": "Создание услуги (auth)", "security": bearerSec, "responses": map[string]interface{}{"201": map[string]interface{}{"description": "Услуга создана"}, "401": map[string]interface{}{"description": "Требуется авторизация"}}},
			},
			"/api/iss-draft": map[string]interface{}{"get": map[string]interface{}{"summary": "Черновик текущего пользователя", "security": bearerSec, "responses": map[string]interface{}{"200": map[string]interface{}{"description": "Черновик"}, "401": map[string]interface{}{"description": "Требуется авторизация"}}}},
			"/api/iss-positions": map[string]interface{}{"get": map[string]interface{}{
				"summary": "Список заявок (creator: только свои, moderator: все)", "security": bearerSec,
				"responses": map[string]interface{}{"200": map[string]interface{}{"description": "Список заявок"}, "401": map[string]interface{}{"description": "Требуется авторизация"}},
			}},
			"/api/iss-positions/{id}/complete": map[string]interface{}{
				"put": map[string]interface{}{
					"summary":  "Завершение/отклонение заявки (только moderator)",
					"security": bearerSec,
					"requestBody": rb([]string{"status"}, map[string]interface{}{"status": map[string]interface{}{"type": "string", "enum": []string{"completed", "rejected"}}}, map[string]interface{}{"status": "completed"}),
					"responses": map[string]interface{}{"200": map[string]interface{}{"description": "Статус обновлен"}, "403": map[string]interface{}{"description": "Только для модератора"}, "401": map[string]interface{}{"description": "Требуется авторизация"}},
				},
			},
		},
	}
	writeJSON(w, http.StatusOK, spec)
}

func writeAuthLoginResponse(w http.ResponseWriter, u authUser, token string) {
	var resp loginResponse
	resp.Token = token
	resp.Role = u.Role
	resp.User.ID = u.ID
	resp.User.Username = u.Username
	writeJSON(w, http.StatusOK, resp)
}
