package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
)

var (
	db        *sql.DB
	templates map[string]*template.Template
)

const defaultUserID = 1 // Пользователь по умолчанию (Иванов)

// ServiceFilters — фильтры каталога услуг для HTML и REST.
// Поддерживаем несколько параметров, чтобы фронтенд мог передавать фильтры в query.
type ServiceFilters struct {
	Search       string
	Country      string
	Timezone     string
	MinElevation *int
	MaxElevation *int
}

func main() {
	// Подключение к PostgreSQL
	dsn := getEnv("DATABASE_URL", "postgres://root:root@localhost:5433/RIP?sslmode=disable")
	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatalf("БД недоступна: %v", err)
	}
	fmt.Println("Подключение к PostgreSQL установлено")

	// Выполнить миграцию
	runMigration()
	ensureAuthSchema()

	// Парсинг шаблонов
	templates = make(map[string]*template.Template)
	templates["services"] = template.Must(
		template.ParseFiles("templates/base.html", "templates/services.html"))
	templates["service_detail"] = template.Must(
		template.ParseFiles("templates/base.html", "templates/service_detail.html"))
	templates["iss_position"] = template.Must(
		template.ParseFiles("templates/base.html", "templates/iss_position.html"))

	mux := http.NewServeMux()

	// Статика
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// API-маршруты (REST для SPA)
	registerAPIHandlers(mux)

	// Маршруты HTML: сначала префиксы, "/" — последним (явный fallback)
	mux.HandleFunc("/services/", servicesRouter)            // GET: детальная + POST: добавить в заявку
	mux.HandleFunc("/iss-positions/", issPositionsRouter)  // GET: просмотр заявки + POST: удалить заявку
	mux.HandleFunc("/", servicesListHandler)               // GET: список услуг и неизвестные пути

	port := getEnv("PORT", "8080")
	printListenAddresses(port)
	handler := corsMiddleware(mux)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func runMigration() {
	for _, file := range []string{"migrations/001_init.sql", "migrations/002_lab_index.sql"} {
		data, err := os.ReadFile(file)
		if err != nil {
			log.Printf("Миграция %s не найдена: %v", file, err)
			continue
		}
		if _, err := db.Exec(string(data)); err != nil {
			log.Printf("Ошибка миграции %s (может быть уже выполнена): %v", file, err)
		} else {
			fmt.Printf("Миграция %s выполнена успешно\n", file)
		}
	}
}

// ============================================================
// ORM-функции (работа с БД через database/sql)
// ============================================================

const (
	defaultServicesPageSize = 20
	maxServicesPageSize     = 100
)

func buildActivePointsWhere(filters ServiceFilters) (string, []interface{}) {
	where := ` FROM observation_points WHERE status = 'active'`
	var args []interface{}
	nextArg := func() string {
		return "$" + strconv.Itoa(len(args)+1)
	}

	if filters.Search != "" {
		arg := "%" + strings.ToLower(filters.Search) + "%"
		p := nextArg()
		// Выражение совпадает с GIN-индексом idx_obs_points_search_trgm (pg_trgm)
		where += ` AND (lower(name) || ' ' || lower(country)) LIKE ` + p
		args = append(args, arg)
	}
	if filters.Country != "" {
		p := nextArg()
		where += ` AND LOWER(country) = ` + p
		args = append(args, strings.ToLower(filters.Country))
	}
	if filters.Timezone != "" {
		p := nextArg()
		where += ` AND LOWER(timezone) = ` + p
		args = append(args, strings.ToLower(filters.Timezone))
	}
	if filters.MinElevation != nil {
		p := nextArg()
		where += ` AND elevation >= ` + p
		args = append(args, *filters.MinElevation)
	}
	if filters.MaxElevation != nil {
		p := nextArg()
		where += ` AND elevation <= ` + p
		args = append(args, *filters.MaxElevation)
	}
	return where, args
}

func scanObservationPoints(rows *sql.Rows) ([]ObservationPoint, error) {
	var points []ObservationPoint
	for rows.Next() {
		var p ObservationPoint
		if err := rows.Scan(&p.ID, &p.Name, &p.Country, &p.Latitude, &p.Longitude,
			&p.Elevation, &p.Timezone, &p.BestTime, &p.LightPollution,
			&p.WeatherConditions, &p.Description, &p.ImageURL, &p.VideoURL, &p.Status); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

// getActivePoints — получить активные точки наблюдения с фильтрами (ORM)
func getActivePoints(filters ServiceFilters) ([]ObservationPoint, error) {
	where, args := buildActivePointsWhere(filters)
	query := `SELECT id, name, country, latitude, longitude, elevation, timezone,
	          best_time, light_pollution, weather_conditions, description,
	          image_url, video_url, status` + where + ` ORDER BY id`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanObservationPoints(rows)
}

// getActivePointsPaged — список с пагинацией и общим числом записей (доп. задание: индексы)
func getActivePointsPaged(filters ServiceFilters, page, pageSize int) ([]ObservationPoint, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultServicesPageSize
	}
	if pageSize > maxServicesPageSize {
		pageSize = maxServicesPageSize
	}

	where, args := buildActivePointsWhere(filters)

	var total int
	countQuery := `SELECT COUNT(*)` + where
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	selectCols := `SELECT id, name, country, latitude, longitude, elevation, timezone,
	               best_time, light_pollution, weather_conditions, description,
	               image_url, video_url, status`
	limitArg := "$" + strconv.Itoa(len(args)+1)
	offsetArg := "$" + strconv.Itoa(len(args)+2)
	query := selectCols + where + ` ORDER BY id LIMIT ` + limitArg + ` OFFSET ` + offsetArg
	pageArgs := append(append([]interface{}{}, args...), pageSize, (page-1)*pageSize)

	rows, err := db.Query(query, pageArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	points, err := scanObservationPoints(rows)
	if err != nil {
		return nil, 0, err
	}
	return points, total, nil
}

// getActivePointsByIDs — получить активные точки по списку id с сохранением исходного порядка.
func getActivePointsByIDs(ids []int) ([]ObservationPoint, error) {
	if len(ids) == 0 {
		return []ObservationPoint{}, nil
	}

	args := make([]interface{}, 0, len(ids))
	placeholders := make([]string, 0, len(ids))
	unique := make(map[int]struct{}, len(ids))
	orderedUnique := make([]int, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, exists := unique[id]; exists {
			continue
		}
		unique[id] = struct{}{}
		orderedUnique = append(orderedUnique, id)
		args = append(args, id)
		placeholders = append(placeholders, "$"+strconv.Itoa(len(args)))
	}
	if len(orderedUnique) == 0 {
		return []ObservationPoint{}, nil
	}

	query := `SELECT id, name, country, latitude, longitude, elevation, timezone,
	          best_time, light_pollution, weather_conditions, description,
	          image_url, video_url, status
	          FROM observation_points
	          WHERE status = 'active' AND id IN (` + strings.Join(placeholders, ",") + `)`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byID := make(map[int]ObservationPoint, len(orderedUnique))
	for rows.Next() {
		var p ObservationPoint
		if err := rows.Scan(&p.ID, &p.Name, &p.Country, &p.Latitude, &p.Longitude,
			&p.Elevation, &p.Timezone, &p.BestTime, &p.LightPollution,
			&p.WeatherConditions, &p.Description, &p.ImageURL, &p.VideoURL, &p.Status); err != nil {
			return nil, err
		}
		byID[p.ID] = p
	}

	ordered := make([]ObservationPoint, 0, len(orderedUnique))
	for _, id := range orderedUnique {
		if p, ok := byID[id]; ok {
			ordered = append(ordered, p)
		}
	}
	return ordered, nil
}

// getPointByID — получить точку по ID (ORM)
func getPointByID(id int) (*ObservationPoint, error) {
	p := &ObservationPoint{}
	err := db.QueryRow(
		`SELECT id, name, country, latitude, longitude, elevation, timezone,
		 best_time, light_pollution, weather_conditions, description,
		 image_url, video_url, status
		 FROM observation_points WHERE id = $1 AND status = 'active'`, id,
	).Scan(&p.ID, &p.Name, &p.Country, &p.Latitude, &p.Longitude,
		&p.Elevation, &p.Timezone, &p.BestTime, &p.LightPollution,
		&p.WeatherConditions, &p.Description, &p.ImageURL, &p.VideoURL, &p.Status)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// getDraftISSPosition — получить черновик заявки пользователя (ORM)
func getDraftISSPosition(userID int) (*ISSPosition, error) {
	c := &ISSPosition{}
	err := db.QueryRow(
		`SELECT id, status, created_at, creator_id, formed_at, completed_at,
		 moderator_id, observation_date, total_visibility
		 FROM iss_positions WHERE creator_id = $1 AND status = 'draft'`, userID,
	).Scan(&c.ID, &c.Status, &c.CreatedAt, &c.CreatorID, &c.FormedAt,
		&c.CompletedAt, &c.ModeratorID, &c.ObservationDate, &c.TotalVisibility)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// getISSPositionByID — получить заявку по ID (ORM)
func getISSPositionByID(id int) (*ISSPosition, error) {
	c := &ISSPosition{}
	err := db.QueryRow(
		`SELECT id, status, created_at, creator_id, formed_at, completed_at,
		 moderator_id, observation_date, total_visibility
		 FROM iss_positions WHERE id = $1`, id,
	).Scan(&c.ID, &c.Status, &c.CreatedAt, &c.CreatorID, &c.FormedAt,
		&c.CompletedAt, &c.ModeratorID, &c.ObservationDate, &c.TotalVisibility)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// getISSPositionPointsCount — количество точек в заявке (ORM)
func getISSPositionPointsCount(posID int) int {
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM iss_position_points WHERE iss_position_id = $1`, posID).Scan(&count)
	return count
}

// getISSPositionPoints — получить точки в заявке с полными данными (ORM)
func getISSPositionPoints(posID int) ([]EnrichedISSPoint, error) {
	rows, err := db.Query(
		`SELECT op.id, op.name, op.country, op.latitude, op.longitude, op.elevation,
		 op.timezone, op.best_time, op.light_pollution, op.weather_conditions,
		 op.description, op.image_url, op.video_url, op.status,
		 ipp.observation_order, ipp.is_primary, ipp.observer_name,
		 ipp.iss_latitude, ipp.iss_longitude
		 FROM iss_position_points ipp
		 JOIN observation_points op ON op.id = ipp.point_id
		 WHERE ipp.iss_position_id = $1
		 ORDER BY ipp.observation_order`, posID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []EnrichedISSPoint
	for rows.Next() {
		var ep EnrichedISSPoint
		var obsName sql.NullString
		if err := rows.Scan(&ep.ID, &ep.Name, &ep.Country, &ep.Latitude, &ep.Longitude,
			&ep.Elevation, &ep.Timezone, &ep.BestTime, &ep.LightPollution,
			&ep.WeatherConditions, &ep.Description, &ep.ImageURL, &ep.VideoURL, &ep.Status,
			&ep.ObservationOrder, &ep.IsPrimary, &obsName,
			&ep.ISSLatitude, &ep.ISSLongitude); err != nil {
			return nil, err
		}
		if obsName.Valid {
			ep.ObserverName = obsName.String
		}
		result = append(result, ep)
	}
	return result, nil
}

// addPointToISSPosition — добавить точку в заявку (ORM)
// Если черновика нет — создаёт новую заявку
func addPointToISSPosition(userID, pointID int) error {
	// Получить или создать черновик
	pos, err := getDraftISSPosition(userID)
	if err == sql.ErrNoRows {
		// Создать новую заявку-черновик
		var newID int
		err = db.QueryRow(
			`INSERT INTO iss_positions (status, creator_id) VALUES ('draft', $1) RETURNING id`,
			userID,
		).Scan(&newID)
		if err != nil {
			return fmt.Errorf("ошибка создания заявки: %w", err)
		}
		pos = &ISSPosition{ID: newID}
	} else if err != nil {
		return err
	}

	// Определить следующий порядок
	var maxOrder int
	db.QueryRow(`SELECT COALESCE(MAX(observation_order), 0) FROM iss_position_points WHERE iss_position_id = $1`,
		pos.ID).Scan(&maxOrder)

	// Добавить точку (INSERT с ON CONFLICT — составной уникальный ключ)
	_, err = db.Exec(
		`INSERT INTO iss_position_points (iss_position_id, point_id, observation_order)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (iss_position_id, point_id) DO NOTHING`,
		pos.ID, pointID, maxOrder+1)
	return err
}

// deleteISSPositionSQL — логическое удаление заявки через raw SQL UPDATE (без ORM)
func deleteISSPositionSQL(posID int) error {
	// Используем raw SQL, как требуется в задании (без ORM)
	result, err := db.Exec(
		`UPDATE iss_positions SET status = 'deleted' WHERE id = $1 AND status = 'draft'`, posID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("заявка не найдена или уже удалена")
	}
	return nil
}

// ============================================================
// HTTP-обработчики (контроллеры)
// ============================================================

// redirectHome — редирект на главную: Location + HTML (meta refresh / script) на случай кэша или старых клиентов
func redirectHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Location", "/")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusFound)
	const page = `<!DOCTYPE html><html lang="ru"><head><meta charset="utf-8">
<meta http-equiv="refresh" content="0;url=/">
<title>Перенаправление</title></head><body>
<script>location.replace("/")</script>
<p><a href="/">Перейти на главную</a></p>
</body></html>`
	fmt.Fprint(w, page)
}

// servicesListHandler — GET /: список услуг с поиском
// (паттерн "/" в net/http также перехватывает неизвестные пути, не совпавшие с более специфичными маршрутами)
func servicesListHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		redirectHome(w, r)
		return
	}

	searchQuery := strings.TrimSpace(r.URL.Query().Get("search"))

	// Получение и поиск услуг через ORM
	points, err := getActivePoints(ServiceFilters{Search: searchQuery})
	if err != nil {
		http.Error(w, "Ошибка получения данных: "+err.Error(), 500)
		return
	}

	// Текущая заявка-черновик (корзина)
	draft, _ := getDraftISSPosition(defaultUserID)
	var posID, posCount int
	var hasCart bool
	if draft != nil {
		posID = draft.ID
		posCount = getISSPositionPointsCount(draft.ID)
		hasCart = true
	}

	data := map[string]interface{}{
		"Services":    points,
		"SearchQuery": searchQuery,
		"PosID":       posID,
		"PosCount":    posCount,
		"HasCart":     hasCart,
		"ShowCart":    true,
		"ShowSearch":  true,
	}

	if err := templates["services"].ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

// servicesRouter — роутер /services/<id>/
func servicesRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/services/")
	path = strings.TrimSuffix(path, "/")

	if path == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Проверяем, не является ли это POST на /services/<id>/add/
	parts := strings.Split(path, "/")
	if len(parts) == 2 && parts[1] == "add" {
		if r.Method != http.MethodPost {
			redirectHome(w, r)
			return
		}
		id, err := strconv.Atoi(parts[0])
		if err != nil {
			redirectHome(w, r)
			return
		}
		addToISSPositionHandler(w, r, id)
		return
	}

	if len(parts) != 1 {
		redirectHome(w, r)
		return
	}

	id, err := strconv.Atoi(parts[0])
	if err != nil {
		redirectHome(w, r)
		return
	}

	serviceDetailHandler(w, r, id)
}

// serviceDetailHandler — GET /services/<id>/: детальная страница (Vibes)
func serviceDetailHandler(w http.ResponseWriter, r *http.Request, pointID int) {
	point, err := getPointByID(pointID)
	if err != nil {
		redirectHome(w, r)
		return
	}

	draft, _ := getDraftISSPosition(defaultUserID)
	var posID, posCount int
	if draft != nil {
		posID = draft.ID
		posCount = getISSPositionPointsCount(draft.ID)
	}

	data := map[string]interface{}{
		"Service":    point,
		"PosID":      posID,
		"PosCount":   posCount,
		"ShowCart":   false,
		"ShowSearch": false,
	}

	if err := templates["service_detail"].ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

// addToISSPositionHandler — POST /services/<id>/add/: добавить услугу в заявку (ORM)
func addToISSPositionHandler(w http.ResponseWriter, r *http.Request, pointID int) {
	if err := addPointToISSPosition(defaultUserID, pointID); err != nil {
		http.Error(w, "Ошибка добавления: "+err.Error(), 500)
		return
	}
	// Редирект обратно на страницу услуги
	http.Redirect(w, r, fmt.Sprintf("/services/%d/", pointID), http.StatusSeeOther)
}

// issPositionsRouter — роутер /iss-positions/<id>/
func issPositionsRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/iss-positions/")
	path = strings.TrimSuffix(path, "/")

	parts := strings.Split(path, "/")

	// POST /iss-positions/<id>/delete/ — удаление заявки через SQL
	if len(parts) == 2 && parts[1] == "delete" {
		if r.Method != http.MethodPost {
			redirectHome(w, r)
			return
		}
		id, err := strconv.Atoi(parts[0])
		if err != nil {
			redirectHome(w, r)
			return
		}
		deleteISSPositionHandler(w, r, id)
		return
	}

	if len(parts) != 1 {
		redirectHome(w, r)
		return
	}

	// GET /iss-positions/<id>/ — просмотр заявки
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		redirectHome(w, r)
		return
	}
	issPositionDetailHandler(w, r, id)
}

// issPositionDetailHandler — GET /iss-positions/<id>/: просмотр заявки (ORM)
func issPositionDetailHandler(w http.ResponseWriter, r *http.Request, posID int) {
	pos, err := getISSPositionByID(posID)
	if err != nil {
		redirectHome(w, r)
		return
	}

	// Удалённые заявки просматривать нельзя — редирект на главную
	if pos.Status == "deleted" {
		redirectHome(w, r)
		return
	}

	posPoints, err := getISSPositionPoints(posID)
	if err != nil {
		http.Error(w, "Ошибка получения данных: "+err.Error(), 500)
		return
	}

	// Получить имя создателя
	var creatorName string
	db.QueryRow(`SELECT full_name FROM users WHERE id = $1`, pos.CreatorID).Scan(&creatorName)

	var moderatorName string
	if pos.ModeratorID.Valid {
		db.QueryRow(`SELECT full_name FROM users WHERE id = $1`, pos.ModeratorID.Int64).Scan(&moderatorName)
	}

	data := map[string]interface{}{
		"ISSPosition":   pos,
		"PosServices":   posPoints,
		"CreatorName":   creatorName,
		"ModeratorName": moderatorName,
		"IsDraft":       pos.Status == "draft",
		"ShowCart":      false,
		"ShowSearch":    false,
	}

	if err := templates["iss_position"].ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

// deleteISSPositionHandler — POST /iss-positions/<id>/delete/: логическое удаление через SQL UPDATE
func deleteISSPositionHandler(w http.ResponseWriter, r *http.Request, posID int) {
	if err := deleteISSPositionSQL(posID); err != nil {
		http.Error(w, "Ошибка удаления: "+err.Error(), 400)
		return
	}
	// Редирект на главную
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
