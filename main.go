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

	// Парсинг шаблонов
	templates = make(map[string]*template.Template)
	templates["services"] = template.Must(
		template.ParseFiles("templates/base.html", "templates/services.html"))
	templates["service_detail"] = template.Must(
		template.ParseFiles("templates/base.html", "templates/service_detail.html"))
	templates["calculation"] = template.Must(
		template.ParseFiles("templates/base.html", "templates/calculation.html"))

	// Статика
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Маршруты: 3 GET + 2 POST = 5 HTTP методов
	http.HandleFunc("/", servicesListHandler)             // GET: список услуг
	http.HandleFunc("/services/", servicesRouter)         // GET: детальная + POST: добавить в заявку
	http.HandleFunc("/calculations/", calculationsRouter) // GET: просмотр заявки + POST: удалить заявку

	fmt.Println("Сервер запущен на http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
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
	data, err := os.ReadFile("migrations/001_init.sql")
	if err != nil {
		log.Printf("Миграция не найдена: %v", err)
		return
	}
	if _, err := db.Exec(string(data)); err != nil {
		log.Printf("Ошибка миграции (может быть уже выполнена): %v", err)
	} else {
		fmt.Println("Миграция выполнена успешно")
	}
}

// ============================================================
// ORM-функции (работа с БД через database/sql)
// ============================================================

// getActivePoints — получить все активные точки наблюдения (ORM)
func getActivePoints(search string) ([]ObservationPoint, error) {
	query := `SELECT id, name, country, latitude, longitude, elevation, timezone,
	          best_time, light_pollution, weather_conditions, description,
	          image_url, video_url, status
	          FROM observation_points WHERE status = 'active'`
	var args []interface{}

	if search != "" {
		query += ` AND (LOWER(name) LIKE $1 OR LOWER(country) LIKE $1)`
		args = append(args, "%"+strings.ToLower(search)+"%")
	}
	query += ` ORDER BY id`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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
	return points, nil
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

// getDraftCalculation — получить черновик заявки пользователя (ORM)
func getDraftCalculation(userID int) (*Calculation, error) {
	c := &Calculation{}
	err := db.QueryRow(
		`SELECT id, status, created_at, creator_id, formed_at, completed_at,
		 moderator_id, observation_date, total_visibility
		 FROM calculations WHERE creator_id = $1 AND status = 'draft'`, userID,
	).Scan(&c.ID, &c.Status, &c.CreatedAt, &c.CreatorID, &c.FormedAt,
		&c.CompletedAt, &c.ModeratorID, &c.ObservationDate, &c.TotalVisibility)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// getCalculationByID — получить заявку по ID (ORM)
func getCalculationByID(id int) (*Calculation, error) {
	c := &Calculation{}
	err := db.QueryRow(
		`SELECT id, status, created_at, creator_id, formed_at, completed_at,
		 moderator_id, observation_date, total_visibility
		 FROM calculations WHERE id = $1`, id,
	).Scan(&c.ID, &c.Status, &c.CreatedAt, &c.CreatorID, &c.FormedAt,
		&c.CompletedAt, &c.ModeratorID, &c.ObservationDate, &c.TotalVisibility)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// getCalcPointsCount — количество точек в заявке (ORM)
func getCalcPointsCount(calcID int) int {
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM calculation_points WHERE calculation_id = $1`, calcID).Scan(&count)
	return count
}

// getCalcPoints — получить точки в заявке с полными данными (ORM)
func getCalcPoints(calcID int) ([]EnrichedCalcPoint, error) {
	rows, err := db.Query(
		`SELECT op.id, op.name, op.country, op.latitude, op.longitude, op.elevation,
		 op.timezone, op.best_time, op.light_pollution, op.weather_conditions,
		 op.description, op.image_url, op.video_url, op.status,
		 cp.observation_order, cp.is_primary, cp.observer_name, cp.position_result
		 FROM calculation_points cp
		 JOIN observation_points op ON op.id = cp.point_id
		 WHERE cp.calculation_id = $1
		 ORDER BY cp.observation_order`, calcID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []EnrichedCalcPoint
	for rows.Next() {
		var ep EnrichedCalcPoint
		var obsName, posResult sql.NullString
		if err := rows.Scan(&ep.ID, &ep.Name, &ep.Country, &ep.Latitude, &ep.Longitude,
			&ep.Elevation, &ep.Timezone, &ep.BestTime, &ep.LightPollution,
			&ep.WeatherConditions, &ep.Description, &ep.ImageURL, &ep.VideoURL, &ep.Status,
			&ep.ObservationOrder, &ep.IsPrimary, &obsName, &posResult); err != nil {
			return nil, err
		}
		if obsName.Valid {
			ep.ObserverName = obsName.String
		}
		if posResult.Valid {
			ep.PositionResult = posResult.String
		}
		result = append(result, ep)
	}
	return result, nil
}

// addPointToCalculation — добавить точку в заявку (ORM)
// Если черновика нет — создаёт новую заявку
func addPointToCalculation(userID, pointID int) error {
	// Получить или создать черновик
	calc, err := getDraftCalculation(userID)
	if err == sql.ErrNoRows {
		// Создать новую заявку-черновик
		var newID int
		err = db.QueryRow(
			`INSERT INTO calculations (status, creator_id) VALUES ('draft', $1) RETURNING id`,
			userID,
		).Scan(&newID)
		if err != nil {
			return fmt.Errorf("ошибка создания заявки: %w", err)
		}
		calc = &Calculation{ID: newID}
	} else if err != nil {
		return err
	}

	// Определить следующий порядок
	var maxOrder int
	db.QueryRow(`SELECT COALESCE(MAX(observation_order), 0) FROM calculation_points WHERE calculation_id = $1`,
		calc.ID).Scan(&maxOrder)

	// Добавить точку (INSERT с ON CONFLICT — составной уникальный ключ)
	_, err = db.Exec(
		`INSERT INTO calculation_points (calculation_id, point_id, observation_order)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (calculation_id, point_id) DO NOTHING`,
		calc.ID, pointID, maxOrder+1)
	return err
}

// deleteCalculationSQL — логическое удаление заявки через raw SQL UPDATE (без ORM)
func deleteCalculationSQL(calcID int) error {
	// Используем raw SQL, как требуется в задании (без ORM)
	result, err := db.Exec(
		`UPDATE calculations SET status = 'deleted' WHERE id = $1 AND status = 'draft'`, calcID)
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

// servicesListHandler — GET /: список услуг с поиском
func servicesListHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	searchQuery := strings.TrimSpace(r.URL.Query().Get("search"))

	// Получение и поиск услуг через ORM
	points, err := getActivePoints(searchQuery)
	if err != nil {
		http.Error(w, "Ошибка получения данных: "+err.Error(), 500)
		return
	}

	// Текущая заявка-черновик (корзина)
	draft, _ := getDraftCalculation(defaultUserID)
	var calcID, calcCount int
	var hasCart bool
	if draft != nil {
		calcID = draft.ID
		calcCount = getCalcPointsCount(draft.ID)
		hasCart = true
	}

	data := map[string]interface{}{
		"Services":    points,
		"SearchQuery": searchQuery,
		"CalcID":      calcID,
		"CalcCount":   calcCount,
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
	if len(parts) == 2 && parts[1] == "add" && r.Method == http.MethodPost {
		id, err := strconv.Atoi(parts[0])
		if err != nil {
			http.NotFound(w, r)
			return
		}
		addToCalculationHandler(w, r, id)
		return
	}

	id, err := strconv.Atoi(parts[0])
	if err != nil {
		http.NotFound(w, r)
		return
	}

	serviceDetailHandler(w, r, id)
}

// serviceDetailHandler — GET /services/<id>/: детальная страница (Vibes)
func serviceDetailHandler(w http.ResponseWriter, r *http.Request, pointID int) {
	point, err := getPointByID(pointID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	draft, _ := getDraftCalculation(defaultUserID)
	var calcID, calcCount int
	if draft != nil {
		calcID = draft.ID
		calcCount = getCalcPointsCount(draft.ID)
	}

	data := map[string]interface{}{
		"Service":    point,
		"CalcID":     calcID,
		"CalcCount":  calcCount,
		"ShowCart":   false,
		"ShowSearch": false,
	}

	if err := templates["service_detail"].ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

// addToCalculationHandler — POST /services/<id>/add/: добавить услугу в заявку (ORM)
func addToCalculationHandler(w http.ResponseWriter, r *http.Request, pointID int) {
	if err := addPointToCalculation(defaultUserID, pointID); err != nil {
		http.Error(w, "Ошибка добавления: "+err.Error(), 500)
		return
	}
	// Редирект обратно на страницу услуги
	http.Redirect(w, r, fmt.Sprintf("/services/%d/", pointID), http.StatusSeeOther)
}

// calculationsRouter — роутер /calculations/<id>/
func calculationsRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/calculations/")
	path = strings.TrimSuffix(path, "/")

	parts := strings.Split(path, "/")

	// POST /calculations/<id>/delete/ — удаление заявки через SQL
	if len(parts) == 2 && parts[1] == "delete" && r.Method == http.MethodPost {
		id, err := strconv.Atoi(parts[0])
		if err != nil {
			http.NotFound(w, r)
			return
		}
		deleteCalculationHandler(w, r, id)
		return
	}

	// GET /calculations/<id>/ — просмотр заявки
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		http.NotFound(w, r)
		return
	}
	calculationDetailHandler(w, r, id)
}

// calculationDetailHandler — GET /calculations/<id>/: просмотр заявки (ORM)
func calculationDetailHandler(w http.ResponseWriter, r *http.Request, calcID int) {
	calc, err := getCalculationByID(calcID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Удалённые заявки просматривать нельзя
	if calc.Status == "deleted" {
		http.Error(w, "Заявка удалена", http.StatusGone)
		return
	}

	calcPoints, err := getCalcPoints(calcID)
	if err != nil {
		http.Error(w, "Ошибка получения данных: "+err.Error(), 500)
		return
	}

	// Получить имя создателя
	var creatorName string
	db.QueryRow(`SELECT full_name FROM users WHERE id = $1`, calc.CreatorID).Scan(&creatorName)

	var moderatorName string
	if calc.ModeratorID.Valid {
		db.QueryRow(`SELECT full_name FROM users WHERE id = $1`, calc.ModeratorID.Int64).Scan(&moderatorName)
	}

	data := map[string]interface{}{
		"Calculation":   calc,
		"CalcServices":  calcPoints,
		"CreatorName":   creatorName,
		"ModeratorName": moderatorName,
		"IsDraft":       calc.Status == "draft",
		"ShowCart":      false,
		"ShowSearch":    false,
	}

	if err := templates["calculation"].ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

// deleteCalculationHandler — POST /calculations/<id>/delete/: логическое удаление через SQL UPDATE
func deleteCalculationHandler(w http.ResponseWriter, r *http.Request, calcID int) {
	if err := deleteCalculationSQL(calcID); err != nil {
		http.Error(w, "Ошибка удаления: "+err.Error(), 400)
		return
	}
	// Редирект на главную
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
