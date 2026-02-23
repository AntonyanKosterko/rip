package main

import (
	"database/sql"
	"fmt"
	"time"
)

// ObservationPoint — точка наблюдения МКС (услуга)
type ObservationPoint struct {
	ID                int
	Name              string
	Country           string
	Latitude          float64
	Longitude         float64
	Elevation         int
	Timezone          string
	BestTime          sql.NullString
	LightPollution    sql.NullString
	WeatherConditions sql.NullString
	Description       sql.NullString
	ImageURL          sql.NullString
	VideoURL          sql.NullString
	Status            string
}

// Coordinates — строка координат
func (p ObservationPoint) Coordinates() string {
	return fmt.Sprintf("%.4g°, %.4g°", p.Latitude, p.Longitude)
}

// GetImageURL — безопасное получение URL изображения
func (p ObservationPoint) GetImageURL() string {
	if p.ImageURL.Valid {
		return p.ImageURL.String
	}
	return ""
}

// GetVideoURL — безопасное получение URL видео
func (p ObservationPoint) GetVideoURL() string {
	if p.VideoURL.Valid {
		return p.VideoURL.String
	}
	return ""
}

// GetDescription — безопасное получение описания
func (p ObservationPoint) GetDescription() string {
	if p.Description.Valid {
		return p.Description.String
	}
	return ""
}

// GetBestTime — безопасное получение лучшего времени
func (p ObservationPoint) GetBestTime() string {
	if p.BestTime.Valid {
		return p.BestTime.String
	}
	return ""
}

// GetLightPollution — безопасное получение засветки
func (p ObservationPoint) GetLightPollution() string {
	if p.LightPollution.Valid {
		return p.LightPollution.String
	}
	return ""
}

// GetWeatherConditions — безопасное получение погоды
func (p ObservationPoint) GetWeatherConditions() string {
	if p.WeatherConditions.Valid {
		return p.WeatherConditions.String
	}
	return ""
}

// Calculation — заявка (расчёт видимости МКС)
type Calculation struct {
	ID              int
	Status          string
	CreatedAt       time.Time
	CreatorID       int
	FormedAt        sql.NullTime
	CompletedAt     sql.NullTime
	ModeratorID     sql.NullInt64
	ObservationDate sql.NullTime
	TotalVisibility sql.NullString
}

// StatusRu — русское название статуса
func (c Calculation) StatusRu() string {
	switch c.Status {
	case "draft":
		return "Черновик"
	case "deleted":
		return "Удалён"
	case "formed":
		return "Сформирован"
	case "completed":
		return "Завершён"
	case "rejected":
		return "Отклонён"
	default:
		return c.Status
	}
}

// GetObservationDate — форматированная дата наблюдения
func (c Calculation) GetObservationDate() string {
	if c.ObservationDate.Valid {
		return c.ObservationDate.Time.Format("02.01.2006")
	}
	return "—"
}

// GetCreatedAt — форматированная дата создания
func (c Calculation) GetCreatedAt() string {
	return c.CreatedAt.Format("02.01.2006 15:04")
}

// GetFormedAt — форматированная дата формирования
func (c Calculation) GetFormedAt() string {
	if c.FormedAt.Valid {
		return c.FormedAt.Time.Format("02.01.2006 15:04")
	}
	return "—"
}

// GetCompletedAt — форматированная дата завершения
func (c Calculation) GetCompletedAt() string {
	if c.CompletedAt.Valid {
		return c.CompletedAt.Time.Format("02.01.2006 15:04")
	}
	return "—"
}

// GetTotalVisibility — результат расчёта
func (c Calculation) GetTotalVisibility() string {
	if c.TotalVisibility.Valid {
		return c.TotalVisibility.String
	}
	return "Расчёт не выполнен"
}

// CalculationPoint — связь м-м: точка наблюдения в заявке
type CalculationPoint struct {
	ID               int
	CalculationID    int
	PointID          int
	ObservationOrder int
	IsPrimary        bool
	ObserverName     sql.NullString
	PositionResult   sql.NullString
}

// EnrichedCalcPoint — точка в заявке с полными данными
type EnrichedCalcPoint struct {
	ObservationPoint
	ObservationOrder int
	IsPrimary        bool
	ObserverName     string
	PositionResult   string
}

// User — пользователь
type User struct {
	ID       int
	Username string
	FullName string
	Email    sql.NullString
}
