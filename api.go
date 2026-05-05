package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"golang.org/x/crypto/bcrypt"
)

// ============================================================
// Общие утилиты для API
// ============================================================

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		if err := json.NewEncoder(w).Encode(v); err != nil {
			log.Printf("encode json error: %v", err)
		}
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

// ============================================================
// Singleton-пользователи (создатель и модератор)
// ============================================================

var (
	currentUserOnce  sync.Once
	currentUser      *User
	moderatorUser    *User
	currentUserError error
)

func initCurrentUsers() {
	// В данной лабораторной пользователи зафиксированы:
	// creator_id = 1 (ivanov), moderator_id = 2 (petrova)
	currentUser = &User{ID: 1, Username: "ivanov"}
	moderatorUser = &User{ID: 2, Username: "petrova"}
}

func GetCurrentUser() *User {
	currentUserOnce.Do(initCurrentUsers)
	return currentUser
}

func GetModeratorUser() *User {
	currentUserOnce.Do(initCurrentUsers)
	return moderatorUser
}

// ============================================================
// MinIO: загрузка файлов
// ============================================================

var (
	minioOnce   sync.Once
	minioClient *minio.Client
	minioErr    error
)

func getMinioClient() (*minio.Client, error) {
	minioOnce.Do(func() {
		endpoint := getEnv("MINIO_ENDPOINT", "localhost:9000")
		accessKey := getEnv("MINIO_ACCESS_KEY", "minioadmin")
		secretKey := getEnv("MINIO_SECRET_KEY", "minioadmin")
		useSSL := false

		minioClient, minioErr = minio.New(endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
			Secure: useSSL,
		})
	})
	return minioClient, minioErr
}

func uploadToMinio(ctx context.Context, objectName string, file multipart.File, contentType string) (string, error) {
	defer file.Close()

	client, err := getMinioClient()
	if err != nil {
		return "", fmt.Errorf("minio client error: %w", err)
	}

	bucket := getEnv("MINIO_BUCKET", "iss-bucket")

	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return "", fmt.Errorf("bucket check error: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return "", fmt.Errorf("make bucket error: %w", err)
		}
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("read file error: %w", err)
	}

	reader := bytes.NewReader(data)
	size := int64(len(data))

	_, err = client.PutObject(ctx, bucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("put object error: %w", err)
	}

	publicBase := getEnv("MINIO_PUBLIC_URL", "http://localhost:9000")
	return fmt.Sprintf("%s/%s/%s", publicBase, bucket, objectName), nil
}

// ============================================================
// DTO: услуги
// ============================================================

type ServiceResponse struct {
	ID                int     `json:"id"`
	Name              string  `json:"name"`
	Country           string  `json:"country"`
	Latitude          float64 `json:"latitude"`
	Longitude         float64 `json:"longitude"`
	Elevation         int     `json:"elevation"`
	Timezone          string  `json:"timezone"`
	BestTime          string  `json:"best_time,omitempty"`
	LightPollution    string  `json:"light_pollution,omitempty"`
	WeatherConditions string  `json:"weather_conditions,omitempty"`
	Description       string  `json:"description,omitempty"`
	ImageURL          string  `json:"image_url,omitempty"`
	VideoURL          string  `json:"video_url,omitempty"`
}

func serviceToResponse(p ObservationPoint) ServiceResponse {
	return ServiceResponse{
		ID:                p.ID,
		Name:              p.Name,
		Country:           p.Country,
		Latitude:          p.Latitude,
		Longitude:         p.Longitude,
		Elevation:         p.Elevation,
		Timezone:          p.Timezone,
		BestTime:          p.GetBestTime(),
		LightPollution:    p.GetLightPollution(),
		WeatherConditions: p.GetWeatherConditions(),
		Description:       p.GetDescription(),
		ImageURL:          p.GetImageURL(),
		VideoURL:          p.GetVideoURL(),
	}
}

// ============================================================
// DTO: заявки и m-m
// ============================================================

// ISSDraftInfoResponse — текущий черновик заявки на фиксацию положения МКС (id и число точек наблюдения)
type ISSDraftInfoResponse struct {
	ID    *int `json:"id"`
	Count int  `json:"count"`
}

type ISSPositionListItem struct {
	ID       int    `json:"id"`
	Status   string `json:"status"`
	StatusRu string `json:"status_ru"`
	// ThemeRu — тема проекта (лаба): определение положения МКС
	ThemeRu string `json:"theme_ru"`
	// ObservationPointsCount — всего точек наблюдения в заявке
	ObservationPointsCount int `json:"observation_points_count"`
	// ISSPositionDeterminationResultsCount — число точек, где зафиксировано положение МКС (iss_latitude + iss_longitude); результат по теме заявки
	ISSPositionDeterminationResultsCount int `json:"iss_position_determination_results_count"`

	CreatedAt       time.Time  `json:"created_at"`
	FormedAt        *time.Time `json:"formed_at,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	ObservationDate *time.Time `json:"observation_date,omitempty"`
	// TotalVisibility — текст результата расчёта видимости/заявки (всегда в JSON, может быть "")
	TotalVisibility string `json:"total_visibility"`
	CreatorLogin    string `json:"creator_login"`
	ModeratorLogin  string `json:"moderator_login,omitempty"`
}

type ISSPositionPointResponse struct {
	PointID int    `json:"point_id"`
	Name    string `json:"name"`
	Country string `json:"country"`
	// Координаты точки на Земле
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Elevation int     `json:"elevation"`
	Timezone  string  `json:"timezone"`
	// Карточка услуги (как в каталоге) — без omitempty, чтобы ссылки и поля были видны в JSON
	BestTime          string `json:"best_time"`
	LightPollution    string `json:"light_pollution"`
	WeatherConditions string `json:"weather_conditions"`
	Description       string `json:"description"`
	ImageURL          string `json:"image_url"`
	VideoURL          string `json:"video_url"`

	ObservationOrder int    `json:"observation_order"`
	IsPrimary        bool   `json:"is_primary"`
	ObserverName     string `json:"observer_name"`
	ISSLatitude      *float64 `json:"iss_latitude,omitempty"`
	ISSLongitude     *float64 `json:"iss_longitude,omitempty"`
}

type ISSPositionDetailResponse struct {
	ID       int    `json:"id"`
	Status   string `json:"status"`
	StatusRu string `json:"status_ru"`
	// ThemeRu — тема: определение положения МКС с точек наблюдения
	ThemeRu string `json:"theme_ru"`

	CreatedAt   time.Time `json:"created_at"`
	CreatorID   int       `json:"creator_id"`
	CreatorName string    `json:"creator_name"`

	FormedAt        *time.Time `json:"formed_at,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	ModeratorID     *int       `json:"moderator_id,omitempty"`
	ModeratorName   string     `json:"moderator_name,omitempty"`
	ObservationDate *time.Time `json:"observation_date,omitempty"`

	// TotalVisibility — результат расчёта / итоговый текст заявки (всегда ключ в JSON)
	TotalVisibility string `json:"total_visibility"`

	// ObservationPointsCount — число точек в заявке
	ObservationPointsCount int `json:"observation_points_count"`
	// ISSPositionDeterminationResultsCount — сколько точек с зафиксированным положением МКС (по теме заявки)
	ISSPositionDeterminationResultsCount int `json:"iss_position_determination_results_count"`

	Points []ISSPositionPointResponse `json:"points"`
}

type ISSPositionUpdateRequest struct {
	ObservationDate *string `json:"observation_date,omitempty"` // формат YYYY-MM-DD
}

// ISSDraftAddPointRequest — добавить точку наблюдения в черновик заявки
type ISSDraftAddPointRequest struct {
	PointID int `json:"point_id"`
}

type PositionPointUpdateRequest struct {
	ObservationOrder *int     `json:"observation_order,omitempty"`
	IsPrimary        *bool    `json:"is_primary,omitempty"`
	ObserverName     *string  `json:"observer_name,omitempty"`
	ISSLatitude      *float64 `json:"iss_latitude,omitempty"`
	ISSLongitude     *float64 `json:"iss_longitude,omitempty"`
}

// ============================================================
// DTO: пользователь / аутентификация
// ============================================================

type RegisterUserRequest struct {
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Email    string `json:"email,omitempty"`
}

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthResponse — лаба 3: без токена; «текущий пользователь» задаётся singleton GetCurrentUser()
type AuthResponse struct {
	Success bool `json:"success"`
}

// ============================================================
// Регистрация API-маршрутов
// ============================================================

func registerAPIHandlers(mux *http.ServeMux) {
	// Swagger (ЛР4)
	mux.HandleFunc("/swagger", swaggerUIHandler)
	mux.HandleFunc("/swagger/", swaggerUIHandler)
	mux.HandleFunc("/openapi.json", openAPISpecHandler)

	// Услуги
	mux.HandleFunc("/api/services", servicesAPIHandler)
	mux.HandleFunc("/api/services/", serviceByIDAPIHandler)

	// Черновик заявки (МКС): точки наблюдения перед формированием заявки
	mux.HandleFunc("/api/iss-draft/points", issDraftPointsAPIHandler)
	mux.HandleFunc("/api/iss-draft", issDraftInfoAPIHandler)
	// ЛР6: иконка корзины (без авторизации, ответ 200)
	mux.HandleFunc("/api/cart-icon", cartIconAPIHandler)
	mux.HandleFunc("/api/iss-positions/", issPositionsAPIRouter)

	// Заявки (список)
	mux.HandleFunc("/api/iss-positions", issPositionsListAPIHandler)

	// Пользователи и аутентификация (заглушки)
	mux.HandleFunc("/api/users/register", registerUserAPIHandler)
	mux.HandleFunc("/api/auth/login", loginAPIHandler)
	mux.HandleFunc("/api/auth/logout", logoutAPIHandler)
}

// cartIconAPIHandler — GET /api/cart-icon
// Требование ЛР6: запрос на иконку корзины без авторизации, но с ответом 200.
// Для демонстрации возвращаем информацию о черновике пользователя по умолчанию (id=1).
func cartIconAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}

	draft, err := getDraftISSPosition(defaultUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusOK, ISSDraftInfoResponse{ID: nil, Count: 0})
			return
		}
		// Всё равно 200, чтобы фронт не падал (по заданию важен 200)
		writeJSON(w, http.StatusOK, ISSDraftInfoResponse{ID: nil, Count: 0})
		return
	}

	count := getISSPositionPointsCount(draft.ID)
	id := draft.ID
	writeJSON(w, http.StatusOK, ISSDraftInfoResponse{ID: &id, Count: count})
}

// ============================================================
// Хендлеры: услуги
// ============================================================

func servicesAPIHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetServices(w, r)
	case http.MethodPost:
		if _, ok := requireAuth(w, r); !ok {
			return
		}
		handleCreateService(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
	}
}

func handleGetServices(w http.ResponseWriter, r *http.Request) {
	guestID := ensureGuestSession(w, r)
	viewedOnly := r.URL.Query().Get("viewed") == "1" || strings.EqualFold(r.URL.Query().Get("viewed"), "true")
	recentLimit := 6
	if s := strings.TrimSpace(r.URL.Query().Get("limit")); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v <= 0 {
			writeError(w, http.StatusBadRequest, "Некорректный limit")
			return
		}
		recentLimit = v
	}
	excludeID := 0
	if s := strings.TrimSpace(r.URL.Query().Get("exclude_id")); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v <= 0 {
			writeError(w, http.StatusBadRequest, "Некорректный exclude_id")
			return
		}
		excludeID = v
	}

	if viewedOnly {
		ids := getRecentlyViewedServiceIDs(r.Context(), guestID, recentLimit+1)
		if len(ids) == 0 {
			writeJSON(w, http.StatusOK, []ServiceResponse{})
			return
		}
		filteredIDs := make([]int, 0, len(ids))
		for _, id := range ids {
			if id == excludeID {
				continue
			}
			filteredIDs = append(filteredIDs, id)
		}
		if len(filteredIDs) == 0 {
			writeJSON(w, http.StatusOK, []ServiceResponse{})
			return
		}

		points, err := getActivePointsByIDs(filteredIDs)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Ошибка получения данных: "+err.Error())
			return
		}

		resp := make([]ServiceResponse, 0, len(points))
		for _, p := range points {
			resp = append(resp, serviceToResponse(p))
			if len(resp) >= recentLimit {
				break
			}
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}

	filters := ServiceFilters{
		Search:   strings.TrimSpace(r.URL.Query().Get("search")),
		Country:  strings.TrimSpace(r.URL.Query().Get("country")),
		Timezone: strings.TrimSpace(r.URL.Query().Get("timezone")),
	}
	if s := strings.TrimSpace(r.URL.Query().Get("min_elevation")); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Некорректный min_elevation")
			return
		}
		filters.MinElevation = &v
	}
	if s := strings.TrimSpace(r.URL.Query().Get("max_elevation")); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Некорректный max_elevation")
			return
		}
		filters.MaxElevation = &v
	}

	points, err := getActivePoints(filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Ошибка получения данных: "+err.Error())
		return
	}

	resp := make([]ServiceResponse, 0, len(points))
	for _, p := range points {
		resp = append(resp, serviceToResponse(p))
	}
	writeJSON(w, http.StatusOK, resp)
}

func handleCreateService(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "Некорректная форма: "+err.Error())
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		writeError(w, http.StatusBadRequest, "Поле name обязательно")
		return
	}

	country := strings.TrimSpace(r.FormValue("country"))
	latStr := strings.TrimSpace(r.FormValue("latitude"))
	lonStr := strings.TrimSpace(r.FormValue("longitude"))
	elevStr := strings.TrimSpace(r.FormValue("elevation"))
	timezone := strings.TrimSpace(r.FormValue("timezone"))
	bestTime := strings.TrimSpace(r.FormValue("best_time"))
	lightPollution := strings.TrimSpace(r.FormValue("light_pollution"))
	weatherConditions := strings.TrimSpace(r.FormValue("weather_conditions"))
	description := strings.TrimSpace(r.FormValue("description"))

	if country == "" || latStr == "" || lonStr == "" || timezone == "" {
		writeError(w, http.StatusBadRequest, "Обязательные поля: country, latitude, longitude, timezone")
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Некорректная широта")
		return
	}
	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Некорректная долгота")
		return
	}
	var elev int
	if elevStr != "" {
		elev, err = strconv.Atoi(elevStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Некорректная высота")
			return
		}
	}

	slug := generateSlug(name)
	nowSuffix := time.Now().UnixNano()

	var imageURL, videoURL sql.NullString

	if file, header, err := r.FormFile("image"); err == nil {
		ext := strings.ToLower(filepath.Ext(header.Filename))
		if ext == "" {
			ext = ".jpg"
		}
		objectName := fmt.Sprintf("%s_%d%s", slug, nowSuffix, ext)
		url, upErr := uploadToMinio(r.Context(), objectName, file, header.Header.Get("Content-Type"))
		if upErr != nil {
			writeError(w, http.StatusInternalServerError, "Ошибка загрузки изображения: "+upErr.Error())
			return
		}
		imageURL = sql.NullString{String: url, Valid: true}
	}

	if file, header, err := r.FormFile("video"); err == nil {
		ext := strings.ToLower(filepath.Ext(header.Filename))
		if ext == "" {
			ext = ".mp4"
		}
		objectName := fmt.Sprintf("%s_video_%d%s", slug, nowSuffix, ext)
		url, upErr := uploadToMinio(r.Context(), objectName, file, header.Header.Get("Content-Type"))
		if upErr != nil {
			writeError(w, http.StatusInternalServerError, "Ошибка загрузки видео: "+upErr.Error())
			return
		}
		videoURL = sql.NullString{String: url, Valid: true}
	}

	// Вставка в БД
	var newID int
	err = db.QueryRow(
		`INSERT INTO observation_points
		 (name, country, latitude, longitude, elevation, timezone,
		  best_time, light_pollution, weather_conditions, description,
		  image_url, video_url, status)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,'active')
		 RETURNING id`,
		name, country, lat, lon, elev, timezone,
		nullIfEmpty(bestTime), nullIfEmpty(lightPollution), nullIfEmpty(weatherConditions), nullIfEmpty(description),
		imageURL, videoURL,
	).Scan(&newID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Ошибка сохранения услуги: "+err.Error())
		return
	}

	point, err := getPointByID(newID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Ошибка чтения созданной услуги: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, serviceToResponse(*point))
}

func nullIfEmpty(s string) sql.NullString {
	if strings.TrimSpace(s) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func generateSlug(name string) string {
	name = strings.ToLower(name)
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			b.WriteRune('-')
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		slug = "service"
	}
	return slug
}

func serviceByIDAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/services/")
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		writeError(w, http.StatusBadRequest, "Не указан id услуги")
		return
	}
	id, err := strconv.Atoi(path)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Некорректный id услуги")
		return
	}
	point, err := getPointByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "Услуга не найдена")
			return
		}
		writeError(w, http.StatusInternalServerError, "Ошибка получения услуги: "+err.Error())
		return
	}
	guestID := ensureGuestSession(w, r)
	recordViewedService(r.Context(), guestID, id)
	writeJSON(w, http.StatusOK, serviceToResponse(*point))
}

// ============================================================
// Хендлеры: черновик заявки (МКС) и м-м
// ============================================================

func issDraftInfoAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	draft, err := getDraftISSPosition(user.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusOK, ISSDraftInfoResponse{ID: nil, Count: 0})
			return
		}
		writeError(w, http.StatusInternalServerError, "Ошибка получения черновика заявки: "+err.Error())
		return
	}

	count := getISSPositionPointsCount(draft.ID)
	id := draft.ID
	writeJSON(w, http.StatusOK, ISSDraftInfoResponse{ID: &id, Count: count})
}

func issDraftPointsAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}
	var req ISSDraftAddPointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	if req.PointID <= 0 {
		writeError(w, http.StatusBadRequest, "point_id обязателен")
		return
	}

	// Проверим, что услуга существует и активна
	if _, err := getPointByID(req.PointID); err != nil {
		writeError(w, http.StatusNotFound, "Услуга не найдена")
		return
	}

	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	if err := addPointToISSPosition(user.ID, req.PointID); err != nil {
		writeError(w, http.StatusInternalServerError, "Ошибка добавления в заявку: "+err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// issPositionsAPIRouter обрабатывает м-м и операции над конкретной заявкой
func issPositionsAPIRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/iss-positions/")
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		writeError(w, http.StatusBadRequest, "Не указан id заявки")
		return
	}

	parts := strings.Split(path, "/")
	// /api/iss-positions/{id}
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			handleGetISSPosition(w, r, parts[0])
		case http.MethodPut:
			handleUpdateISSPosition(w, r, parts[0])
		case http.MethodDelete:
			handleDeleteISSPosition(w, r, parts[0])
		default:
			w.Header().Set("Allow", "GET, PUT, DELETE")
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		}
		return
	}

	if len(parts) == 2 {
		idPart, action := parts[0], parts[1]
		switch action {
		case "form":
			if r.Method != http.MethodPut {
				w.Header().Set("Allow", "PUT")
				writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
				return
			}
			handleFormISSPosition(w, r, idPart)
			return
		case "complete":
			if r.Method != http.MethodPut {
				w.Header().Set("Allow", "PUT")
				writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
				return
			}
			handleCompleteISSPosition(w, r, idPart)
			return
		case "points":
			// /api/iss-positions/{id}/points — в этой лабе не используется
		default:
			// /api/iss-positions/{id}/points/{pointId}
			if action == "points" && len(parts) == 3 {
				// не достигнем сюда из-за len(parts)==2
				return
			}
		}
	}

	if len(parts) == 3 && parts[1] == "points" {
		idPart, pointPart := parts[0], parts[2]
		switch r.Method {
		case http.MethodPut:
			handleUpdatePositionPoint(w, r, idPart, pointPart)
		case http.MethodDelete:
			handleDeletePositionPoint(w, r, idPart, pointPart)
		default:
			w.Header().Set("Allow", "PUT, DELETE")
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		}
		return
	}

	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		redirectHome(w, r)
		return
	}
	writeError(w, http.StatusNotFound, "Маршрут не найден")
}

// themeRUISSProject — формулировка темы лабораторной для API (список и деталка заявок)
const themeRUISSProject = "Определение положения МКС с точек наблюдения на Земле"

// ============================================================
// Хендлеры: заявки (список, деталка, изменения)
// ============================================================

func issPositionsListAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}

	status := strings.TrimSpace(r.URL.Query().Get("status"))
	fromStr := strings.TrimSpace(r.URL.Query().Get("formed_from"))
	toStr := strings.TrimSpace(r.URL.Query().Get("formed_to"))

	var args []interface{}
	query := `SELECT ip.id, ip.status, ip.created_at, ip.formed_at, ip.completed_at,
	          ip.observation_date, ip.total_visibility,
	          u1.username AS creator_login,
	          COALESCE(u2.username, '') AS moderator_login,
	          COALESCE(ptot.points_total, 0) AS observation_points_count,
	          COALESCE(res.iss_coord_cnt, 0) AS iss_position_determination_results_count
	          FROM iss_positions ip
	          JOIN users u1 ON u1.id = ip.creator_id
	          LEFT JOIN users u2 ON u2.id = ip.moderator_id
	          LEFT JOIN (
	              SELECT iss_position_id, COUNT(*) AS points_total
	              FROM iss_position_points
	              GROUP BY iss_position_id
	          ) ptot ON ptot.iss_position_id = ip.id
	          LEFT JOIN (
	              SELECT iss_position_id, COUNT(*) AS iss_coord_cnt
	              FROM iss_position_points
	              WHERE iss_latitude IS NOT NULL AND iss_longitude IS NOT NULL
	              GROUP BY iss_position_id
	          ) res ON res.iss_position_id = ip.id
	          WHERE ip.status NOT IN ('draft','deleted')`

	if status != "" {
		query += ` AND ip.status = $` + strconv.Itoa(len(args)+1)
		args = append(args, status)
	}
	if user.Role != roleModerator {
		query += ` AND ip.creator_id = $` + strconv.Itoa(len(args)+1)
		args = append(args, user.ID)
	}
	if fromStr != "" {
		if _, err := time.Parse("2006-01-02", fromStr); err == nil {
			query += ` AND ip.formed_at >= $` + strconv.Itoa(len(args)+1)
			args = append(args, fromStr)
		}
	}
	if toStr != "" {
		if _, err := time.Parse("2006-01-02", toStr); err == nil {
			query += ` AND ip.formed_at <= $` + strconv.Itoa(len(args)+1)
			args = append(args, toStr)
		}
	}

	query += ` ORDER BY ip.formed_at DESC NULLS LAST, ip.id DESC`

	rows, err := db.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Ошибка выборки заявок: "+err.Error())
		return
	}
	defer rows.Close()

	var items []ISSPositionListItem
	for rows.Next() {
		var it ISSPositionListItem
		var formedAt, completedAt, obsDate sql.NullTime
		var totalVis sql.NullString
		if err := rows.Scan(
			&it.ID, &it.Status, &it.CreatedAt, &formedAt, &completedAt, &obsDate, &totalVis,
			&it.CreatorLogin, &it.ModeratorLogin,
			&it.ObservationPointsCount, &it.ISSPositionDeterminationResultsCount,
		); err != nil {
			writeError(w, http.StatusInternalServerError, "Ошибка чтения строки: "+err.Error())
			return
		}
		it.StatusRu = ISSPosition{Status: it.Status}.StatusRu()
		it.ThemeRu = themeRUISSProject
		if formedAt.Valid {
			t := formedAt.Time
			it.FormedAt = &t
		}
		if completedAt.Valid {
			t := completedAt.Time
			it.CompletedAt = &t
		}
		if obsDate.Valid {
			t := obsDate.Time
			it.ObservationDate = &t
		}
		if totalVis.Valid {
			it.TotalVisibility = totalVis.String
		}
		items = append(items, it)
	}

	writeJSON(w, http.StatusOK, items)
}

func handleGetISSPosition(w http.ResponseWriter, r *http.Request, idStr string) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		redirectHome(w, r)
		return
	}
	pos, err := getISSPositionByID(id)
	if err != nil {
		redirectHome(w, r)
		return
	}
	if pos.Status == "deleted" {
		redirectHome(w, r)
		return
	}
	if !canAccessPosition(user, pos) {
		writeError(w, http.StatusForbidden, "Нет доступа к заявке")
		return
	}
	points, err := getISSPositionPoints(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Ошибка получения точек заявки: "+err.Error())
		return
	}

	var creatorName string
	db.QueryRow(`SELECT full_name FROM users WHERE id = $1`, pos.CreatorID).Scan(&creatorName)

	var moderatorName string
	var moderatorID *int
	if pos.ModeratorID.Valid {
		idVal := int(pos.ModeratorID.Int64)
		moderatorID = &idVal
		db.QueryRow(`SELECT full_name FROM users WHERE id = $1`, pos.ModeratorID.Int64).Scan(&moderatorName)
	}

	resp := ISSPositionDetailResponse{
		ID:          pos.ID,
		Status:      pos.Status,
		StatusRu:    pos.StatusRu(),
		ThemeRu:     themeRUISSProject,
		CreatedAt:   pos.CreatedAt,
		CreatorID:   pos.CreatorID,
		ModeratorID: moderatorID,
		CreatorName: creatorName,
		ModeratorName: moderatorName,
		Points:      make([]ISSPositionPointResponse, 0, len(points)),
	}
	if pos.FormedAt.Valid {
		t := pos.FormedAt.Time
		resp.FormedAt = &t
	}
	if pos.CompletedAt.Valid {
		t := pos.CompletedAt.Time
		resp.CompletedAt = &t
	}
	if pos.ObservationDate.Valid {
		t := pos.ObservationDate.Time
		resp.ObservationDate = &t
	}
	if pos.TotalVisibility.Valid {
		resp.TotalVisibility = pos.TotalVisibility.String
	}

	resp.ObservationPointsCount = len(points)
	for _, p := range points {
		if p.ISSLatitude.Valid && p.ISSLongitude.Valid {
			resp.ISSPositionDeterminationResultsCount++
		}
	}

	for _, p := range points {
		var issLat, issLon *float64
		if p.ISSLatitude.Valid {
			v := p.ISSLatitude.Float64
			issLat = &v
		}
		if p.ISSLongitude.Valid {
			v := p.ISSLongitude.Float64
			issLon = &v
		}
		resp.Points = append(resp.Points, ISSPositionPointResponse{
			PointID:           p.ID,
			Name:              p.Name,
			Country:           p.Country,
			Latitude:          p.Latitude,
			Longitude:         p.Longitude,
			Elevation:         p.Elevation,
			Timezone:          p.Timezone,
			BestTime:          p.GetBestTime(),
			LightPollution:    p.GetLightPollution(),
			WeatherConditions: p.GetWeatherConditions(),
			Description:       p.GetDescription(),
			ImageURL:          p.GetImageURL(),
			VideoURL:          p.GetVideoURL(),
			ObservationOrder: p.ObservationOrder,
			IsPrimary:         p.IsPrimary,
			ObserverName:      p.ObserverName,
			ISSLatitude:       issLat,
			ISSLongitude:      issLon,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

func handleUpdateISSPosition(w http.ResponseWriter, r *http.Request, idStr string) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	var req ISSPositionUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Некорректный id заявки")
		return
	}
	pos, err := getISSPositionByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Заявка не найдена")
		return
	}
	if pos.Status != "draft" && pos.Status != "formed" {
		writeError(w, http.StatusBadRequest, "Редактировать можно только черновик или сформированную заявку")
		return
	}
	if !canAccessPosition(user, pos) {
		writeError(w, http.StatusForbidden, "Нет доступа к заявке")
		return
	}

	// Разрешаем изменять только дату наблюдения
	if req.ObservationDate == nil {
		writeError(w, http.StatusBadRequest, "Нечего изменять")
		return
	}
	newDate, err := time.Parse("2006-01-02", *req.ObservationDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Некорректный формат observation_date, ожидается YYYY-MM-DD")
		return
	}

	if _, err := db.Exec(`UPDATE iss_positions SET observation_date = $1 WHERE id = $2`, newDate, id); err != nil {
		writeError(w, http.StatusInternalServerError, "Ошибка обновления заявки: "+err.Error())
		return
	}
	pos, err = getISSPositionByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Ошибка чтения заявки: "+err.Error())
		return
	}
	handleGetISSPosition(w, r, idStr)
}

func handleDeleteISSPosition(w http.ResponseWriter, r *http.Request, idStr string) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Некорректный id заявки")
		return
	}
	pos, err := getISSPositionByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Заявка не найдена")
		return
	}
	// Разрешаем удалять только черновик текущего пользователя
	if user.Role != roleModerator && (pos.Status != "draft" || pos.CreatorID != user.ID) {
		writeError(w, http.StatusBadRequest, "Удалять можно только собственный черновик")
		return
	}
	if err := deleteISSPositionSQL(id); err != nil {
		writeError(w, http.StatusBadRequest, "Ошибка удаления: "+err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleFormISSPosition(w http.ResponseWriter, r *http.Request, idStr string) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Некорректный id заявки")
		return
	}
	pos, err := getISSPositionByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Заявка не найдена")
		return
	}
	if user.Role != roleModerator && pos.CreatorID != user.ID {
		writeError(w, http.StatusForbidden, "Сформировать заявку может только её создатель")
		return
	}
	if pos.Status != "draft" {
		writeError(w, http.StatusBadRequest, "Сформировать можно только черновик")
		return
	}

	// Обязательные поля: хотя бы одна точка и дата наблюдения
	points, err := getISSPositionPoints(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Ошибка получения точек: "+err.Error())
		return
	}
	if len(points) == 0 {
		writeError(w, http.StatusBadRequest, "Нельзя сформировать пустую заявку")
		return
	}
	if !pos.ObservationDate.Valid {
		writeError(w, http.StatusBadRequest, "Для формирования заявки нужно указать дату наблюдения (PUT /api/iss-positions/{id})")
		return
	}

	// Рассчитать поле total_visibility (как в миграции лабы 2)
	var bestPoint string
	if len(points) > 0 {
		bestPoint = points[0].Name
	}
	summary := fmt.Sprintf(
		"МКС наблюдаема из %d из %d выбранных точек. Суммарное время видимости: %d мин. Оптимальная точка наблюдения: %s.",
		len(points), len(points), len(points)*6, bestPoint,
	)

	if _, err := db.Exec(
		`UPDATE iss_positions
		 SET status = 'formed', formed_at = NOW(), total_visibility = $1
		 WHERE id = $2 AND status = 'draft'`,
		summary, id,
	); err != nil {
		writeError(w, http.StatusInternalServerError, "Ошибка формирования заявки: "+err.Error())
		return
	}
	handleGetISSPosition(w, r, idStr)
}

func handleCompleteISSPosition(w http.ResponseWriter, r *http.Request, idStr string) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	if user.Role != roleModerator {
		writeError(w, http.StatusForbidden, "Завершение заявки доступно только модератору")
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	if req.Status != "completed" && req.Status != "rejected" {
		writeError(w, http.StatusBadRequest, "status должен быть 'completed' или 'rejected'")
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Некорректный id заявки")
		return
	}
	pos, err := getISSPositionByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Заявка не найдена")
		return
	}

	// Демонстрация: если попытаться завершить не сформированную заявку — ошибка
	if pos.Status != "formed" {
		writeError(w, http.StatusBadRequest, "Завершать/отклонять можно только сформированную заявку")
		return
	}

	if _, err := db.Exec(
		`UPDATE iss_positions
		 SET status = $1, moderator_id = $2, completed_at = NOW()
		 WHERE id = $3 AND status = 'formed'`,
		req.Status, user.ID, id,
	); err != nil {
		writeError(w, http.StatusInternalServerError, "Ошибка обновления статуса заявки: "+err.Error())
		return
	}
	handleGetISSPosition(w, r, idStr)
}

func handleUpdatePositionPoint(w http.ResponseWriter, r *http.Request, posStr, pointStr string) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	posID, err := strconv.Atoi(posStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Некорректный id заявки")
		return
	}
	pointID, err := strconv.Atoi(pointStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Некорректный id точки")
		return
	}

	var req PositionPointUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}

	pos, err := getISSPositionByID(posID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Заявка не найдена")
		return
	}
	if pos.Status != "draft" {
		writeError(w, http.StatusBadRequest, "Изменять точки можно только в черновике")
		return
	}
	if user.Role != roleModerator && pos.CreatorID != user.ID {
		writeError(w, http.StatusForbidden, "Нет доступа к заявке")
		return
	}

	// Собираем SET динамически
	var sets []string
	var args []interface{}

	if req.ObservationOrder != nil {
		sets = append(sets, fmt.Sprintf("observation_order = $%d", len(args)+1))
		args = append(args, *req.ObservationOrder)
	}
	if req.IsPrimary != nil {
		sets = append(sets, fmt.Sprintf("is_primary = $%d", len(args)+1))
		args = append(args, *req.IsPrimary)
	}
	if req.ObserverName != nil {
		sets = append(sets, fmt.Sprintf("observer_name = $%d", len(args)+1))
		args = append(args, *req.ObserverName)
	}
	if req.ISSLatitude != nil {
		sets = append(sets, fmt.Sprintf("iss_latitude = $%d", len(args)+1))
		args = append(args, *req.ISSLatitude)
	}
	if req.ISSLongitude != nil {
		sets = append(sets, fmt.Sprintf("iss_longitude = $%d", len(args)+1))
		args = append(args, *req.ISSLongitude)
	}

	if len(sets) == 0 {
		writeError(w, http.StatusBadRequest, "Нечего изменять")
		return
	}

	args = append(args, posID, pointID)
	query := fmt.Sprintf(
		"UPDATE iss_position_points SET %s WHERE iss_position_id = $%d AND point_id = $%d",
		strings.Join(sets, ", "),
		len(args)-1, len(args),
	)
	if _, err := db.Exec(query, args...); err != nil {
		writeError(w, http.StatusInternalServerError, "Ошибка обновления точки заявки: "+err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleDeletePositionPoint(w http.ResponseWriter, r *http.Request, posStr, pointStr string) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	posID, err := strconv.Atoi(posStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Некорректный id заявки")
		return
	}
	pointID, err := strconv.Atoi(pointStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Некорректный id точки")
		return
	}

	pos, err := getISSPositionByID(posID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Заявка не найдена")
		return
	}
	if pos.Status != "draft" {
		writeError(w, http.StatusBadRequest, "Удалять точки можно только в черновике")
		return
	}
	if user.Role != roleModerator && pos.CreatorID != user.ID {
		writeError(w, http.StatusForbidden, "Нет доступа к заявке")
		return
	}

	res, err := db.Exec(
		`DELETE FROM iss_position_points WHERE iss_position_id = $1 AND point_id = $2`,
		posID, pointID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Ошибка удаления точки заявки: "+err.Error())
		return
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		writeError(w, http.StatusNotFound, "Запись м-м не найдена")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================
// Хендлеры: пользователь и аутентификация (заглушки)
// ============================================================

func registerUserAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}
	var req RegisterUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.FullName = strings.TrimSpace(req.FullName)
	req.Email = strings.TrimSpace(req.Email)

	if req.Username == "" || req.FullName == "" {
		writeError(w, http.StatusBadRequest, "Поля username и full_name обязательны")
		return
	}
	if strings.TrimSpace(req.Password) == "" {
		writeError(w, http.StatusBadRequest, "Поле password обязательно")
		return
	}

	passHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Ошибка хэширования пароля")
		return
	}

	var id int
	err = db.QueryRow(
		`INSERT INTO users (username, full_name, email, password_hash)
		 VALUES ($1, $2, NULLIF($3, ''), $4) RETURNING id`,
		req.Username, req.FullName, req.Email, string(passHash),
	).Scan(&id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Ошибка регистрации пользователя: "+err.Error())
		return
	}

	resp := UserResponse{
		ID:       id,
		Username: req.Username,
		FullName: req.FullName,
		Email:    req.Email,
	}
	writeJSON(w, http.StatusCreated, resp)
}

func loginAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}
	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Некорректный JSON")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		writeError(w, http.StatusBadRequest, "username обязателен")
		return
	}

	var id int
	var storedHash sql.NullString
	if err := db.QueryRow(`SELECT id, COALESCE(password_hash, '') FROM users WHERE username = $1`, req.Username).Scan(&id, &storedHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusUnauthorized, "Неверный логин или пароль")
			return
		}
		writeError(w, http.StatusInternalServerError, "Ошибка проверки пользователя: "+err.Error())
		return
	}
	if !storedHash.Valid || strings.TrimSpace(storedHash.String) == "" {
		writeError(w, http.StatusUnauthorized, "Неверный логин или пароль")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(storedHash.String), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "Неверный логин или пароль")
		return
	}

	u := authUser{
		ID:       id,
		Username: req.Username,
		Role:     roleForUser(id, req.Username),
	}
	token, err := generateJWT(u)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Ошибка генерации JWT: "+err.Error())
		return
	}
	writeAuthLoginResponse(w, u, token)
}

func logoutAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}
	tokenStr := extractBearerToken(r)
	if tokenStr == "" {
		writeError(w, http.StatusUnauthorized, "Токен не передан")
		return
	}
	claims, err := parseJWT(tokenStr)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Невалидный токен")
		return
	}
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		ttl = time.Second
	}
	if err := addToBlacklist(r.Context(), tokenStr, ttl); err != nil {
		writeError(w, http.StatusInternalServerError, "Ошибка добавления в blacklist Redis: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "Токен добавлен в blacklist Redis"})
}

