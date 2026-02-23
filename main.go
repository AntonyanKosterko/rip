package main

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
)

var templates map[string]*template.Template

func main() {
	// Парсинг шаблонов: каждая страница парсится вместе с base
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

	// Маршруты
	http.HandleFunc("/", servicesListHandler)
	http.HandleFunc("/services/", servicesRouter)
	http.HandleFunc("/calculations/", calculationDetailHandler)

	fmt.Println("Сервер запущен на http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Ошибка запуска сервера: %v\n", err)
	}
}

// servicesListHandler — Страница 1: список услуг с поиском
func servicesListHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	searchQuery := strings.TrimSpace(r.URL.Query().Get("search"))

	var filtered []Service
	if searchQuery != "" {
		q := strings.ToLower(searchQuery)
		for _, s := range Services {
			if strings.Contains(strings.ToLower(s.Name), q) ||
				strings.Contains(strings.ToLower(s.Country), q) {
				filtered = append(filtered, s)
			}
		}
	} else {
		filtered = make([]Service, len(Services))
		copy(filtered, Services)
	}

	// Обогащаем URL-ами Minio
	enriched := make([]Service, len(filtered))
	for i, s := range filtered {
		enriched[i] = enrichService(&s)
	}

	data := map[string]interface{}{
		"Services":    enriched,
		"SearchQuery": searchQuery,
		"CalcID":      CurrentCalculation.ID,
		"CalcTitle":   CurrentCalculation.Title,
		"CalcCount":   len(CurrentCalculation.Services),
		"ShowCart":    true,
		"ShowSearch":  true,
	}

	if err := templates["services"].ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

// servicesRouter — роутер для /services/<id>/
func servicesRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/services/")
	path = strings.TrimSuffix(path, "/")

	if path == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	id, err := strconv.Atoi(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	serviceDetailHandler(w, r, id)
}

// serviceDetailHandler — Страница 2: детальная страница услуги (Vibes)
func serviceDetailHandler(w http.ResponseWriter, r *http.Request, serviceID int) {
	svc := getServiceByID(serviceID)
	if svc == nil {
		http.NotFound(w, r)
		return
	}

	enriched := enrichService(svc)

	data := map[string]interface{}{
		"Service":    enriched,
		"CalcID":     CurrentCalculation.ID,
		"CalcCount":  len(CurrentCalculation.Services),
		"ShowCart":   false,
		"ShowSearch": false,
	}

	if err := templates["service_detail"].ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

// calculationDetailHandler — Страница 3: просмотр расчёта
func calculationDetailHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/calculations/")
	path = strings.TrimSuffix(path, "/")

	id, err := strconv.Atoi(path)
	if err != nil || id != CurrentCalculation.ID {
		http.NotFound(w, r)
		return
	}

	var calcServices []EnrichedCalcService
	for _, cs := range CurrentCalculation.Services {
		svc := getServiceByID(cs.ServiceID)
		if svc != nil {
			enriched := enrichService(svc)
			calcServices = append(calcServices, EnrichedCalcService{
				Service:          enriched,
				ObservationOrder: cs.ObservationOrder,
				IsPrimary:        cs.IsPrimary,
				ObserverName:     cs.ObserverName,
				PositionResult:   cs.PositionResult,
			})
		}
	}

	data := map[string]interface{}{
		"Calculation":  CurrentCalculation,
		"CalcServices": calcServices,
		"ShowCart":     false,
		"ShowSearch":   false,
	}

	if err := templates["calculation"].ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}
