package main

// Service — точка наблюдения МКС (услуга)
type Service struct {
	ID                int     `json:"id"`
	Name              string  `json:"name"`
	Country           string  `json:"country"`
	Latitude          float64 `json:"latitude"`
	Longitude         float64 `json:"longitude"`
	Elevation         int     `json:"elevation"`
	Timezone          string  `json:"timezone"`
	BestTime          string  `json:"best_time"`
	LightPollution    string  `json:"light_pollution"`
	WeatherConditions string  `json:"weather_conditions"`
	ImageKey          string  `json:"image_key"`
	VideoKey          string  `json:"video_key"`
	Description       string  `json:"description"`
	ImageURL          string  `json:"-"`
	VideoURL          string  `json:"-"`
}

// CalculationService — связь м-м: услуга в расчёте
type CalculationService struct {
	ServiceID        int    `json:"service_id"`
	ObservationOrder int    `json:"observation_order"`
	IsPrimary        bool   `json:"is_primary"`
	ObserverName     string `json:"observer_name"`
	PositionResult   string `json:"position_result"`
}

// Calculation — расчёт видимости МКС (бывшая «заявка»)
type Calculation struct {
	ID                int                  `json:"id"`
	Title             string               `json:"title"`
	ObservationDate   string               `json:"observation_date"`
	Status            string               `json:"status"`
	CreatedAt         string               `json:"created_at"`
	Moderator         string               `json:"moderator"`
	CalculationResult string               `json:"calculation_result"`
	Services          []CalculationService `json:"services"`
}

// EnrichedCalcService — услуга в расчёте с полными данными + полями м-м
type EnrichedCalcService struct {
	Service
	ObservationOrder int
	IsPrimary        bool
	ObserverName     string
	PositionResult   string
}

// Coordinates — возвращает строку координат для шаблона
func (s Service) Coordinates() string {
	return formatFloat(s.Latitude) + "°, " + formatFloat(s.Longitude) + "°"
}

func formatFloat(f float64) string {
	// Простое форматирование числа
	s := ""
	if f < 0 {
		s = "-"
		f = -f
	}
	whole := int(f)
	frac := int((f-float64(whole))*10000 + 0.5)
	result := s + itoa(whole) + "." + padLeft(itoa(frac), 4)
	// Убираем trailing zeros
	for len(result) > 1 && result[len(result)-1] == '0' {
		result = result[:len(result)-1]
	}
	if result[len(result)-1] == '.' {
		result = result[:len(result)-1]
	}
	return result
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

func padLeft(s string, l int) string {
	for len(s) < l {
		s = "0" + s
	}
	return s
}

// ============================================================
// Данные
// ============================================================

var MinioURL = "http://localhost:9000/iss-bucket"

var Services = []Service{
	{
		ID:                1,
		Name:              "Москва",
		Country:           "Россия",
		Latitude:          55.7558,
		Longitude:         37.6173,
		Elevation:         156,
		Timezone:          "UTC+3 (МСК)",
		BestTime:          "Март — апрель, сентябрь — октябрь",
		LightPollution:    "Высокая",
		WeatherConditions: "Переменная облачность, лучшие условия в ясные зимние ночи",
		ImageKey:          "moscow",
		VideoKey:          "moscow_video",
		Description: "Москва — столица Российской Федерации и один из крупнейших мегаполисов мира " +
			"с населением более 13 миллионов человек. Несмотря на высокий уровень световой " +
			"загрязнённости, МКС можно наблюдать невооружённым глазом благодаря её яркости " +
			"(до −3.8 звёздной величины). Наилучшие условия наблюдения складываются в " +
			"переходные сезоны — весной и осенью, когда ночи достаточно тёмные, а угол " +
			"наклона орбиты МКС обеспечивает пролёт над средними широтами. Рекомендуется " +
			"выбирать наблюдательные площадки на возвышенностях — Воробьёвы горы, смотровая " +
			"площадка МГУ или парк «Коломенское» вдали от ярких источников света. " +
			"Станция проходит по небу за 4–6 минут, двигаясь с запада на восток.",
	},
	{
		ID:                2,
		Name:              "Байконур",
		Country:           "Казахстан",
		Latitude:          45.9647,
		Longitude:         63.305,
		Elevation:         90,
		Timezone:          "UTC+6",
		BestTime:          "Круглый год, лучше всего летом",
		LightPollution:    "Низкая",
		WeatherConditions: "Преимущественно ясная погода, сухой континентальный климат",
		ImageKey:          "baikonur",
		VideoKey:          "baikonur_video",
		Description: "Космодром Байконур — первый и крупнейший в мире космодром, расположенный " +
			"в Кызылординской области Казахстана. Именно отсюда 12 апреля 1961 года " +
			"стартовал корабль «Восток-1» с Юрием Гагариным на борту. Благодаря удалённости " +
			"от крупных городов и сухому климату степной зоны, здесь идеальные условия " +
			"для наблюдения за МКС. Световое загрязнение минимально, а ясных ночей " +
			"в году значительно больше, чем в средней полосе России. Наблюдатель может " +
			"отслеживать пролёт станции на протяжении всего видимого участка траектории — " +
			"от горизонта до горизонта. Максимальная высота пролёта МКС над Байконуром " +
			"может достигать 85–88 градусов, что делает эту точку одной из лучших для наблюдений.",
	},
	{
		ID:                3,
		Name:              "Мыс Канаверал",
		Country:           "США",
		Latitude:          28.3922,
		Longitude:         -80.6077,
		Elevation:         3,
		Timezone:          "UTC−5 (EST)",
		BestTime:          "Ноябрь — февраль",
		LightPollution:    "Средняя",
		WeatherConditions: "Субтропический климат, частая облачность летом",
		ImageKey:          "cape_canaveral",
		VideoKey:          "cape_canaveral_video",
		Description: "Мыс Канаверал — легендарная стартовая площадка NASA на атлантическом побережье " +
			"штата Флорида. Здесь расположены Космический центр Кеннеди и база ВВС США, " +
			"откуда осуществлялись запуски программ «Аполлон», Space Shuttle и современных " +
			"ракет SpaceX. Близость к экватору (28° с.ш.) обеспечивает уникальную геометрию " +
			"пролёта МКС — станция проходит практически через зенит, что позволяет наблюдать " +
			"её максимально яркой. Субтропический климат создаёт идеальные условия зимой, " +
			"когда влажность ниже и небо чаще бывает безоблачным. Наблюдения лучше проводить " +
			"с пляжей Cocoa Beach или с площадок Космического центра Кеннеди, где открытый " +
			"горизонт над Атлантическим океаном позволяет отслеживать станцию от восхода до заката.",
	},
	{
		ID:                4,
		Name:              "Куру",
		Country:           "Французская Гвиана",
		Latitude:          5.1594,
		Longitude:         -52.6503,
		Elevation:         8,
		Timezone:          "UTC−3",
		BestTime:          "Август — ноябрь (сухой сезон)",
		LightPollution:    "Низкая",
		WeatherConditions: "Экваториальный климат, высокая влажность, частые дожди",
		ImageKey:          "kourou",
		VideoKey:          "kourou_video",
		Description: "Гвианский космический центр (Centre Spatial Guyanais) в городе Куру — " +
			"основной космодром Европейского космического агентства (ESA). Расположен " +
			"всего в 5 градусах от экватора, что даёт максимальное преимущество при запусках " +
			"на геостационарную орбиту. Для наблюдения за МКС близость к экватору означает, " +
			"что станция проходит точно через зенит с максимальной яркостью и минимальным " +
			"атмосферным поглощением. Однако экваториальный климат создаёт сложности — " +
			"высокая влажность и частые облака. Лучшее время для наблюдений — сухой сезон " +
			"с августа по ноябрь. Тропические джунгли вокруг космодрома обеспечивают " +
			"минимальное световое загрязнение, что компенсирует сложные погодные условия.",
	},
	{
		ID:                5,
		Name:              "Восточный",
		Country:           "Россия",
		Latitude:          51.8844,
		Longitude:         128.334,
		Elevation:         275,
		Timezone:          "UTC+9",
		BestTime:          "Июнь — август",
		LightPollution:    "Очень низкая",
		WeatherConditions: "Континентальный климат, холодные зимы, тёплое лето",
		ImageKey:          "vostochny",
		VideoKey:          "vostochny_video",
		Description: "Космодром Восточный — новейший российский космодром, расположенный в Амурской " +
			"области Дальнего Востока. Строительство началось в 2012 году, первый пуск " +
			"состоялся 28 апреля 2016 года. Космодром призван обеспечить России независимый " +
			"доступ в космос с собственной территории. Окружающая тайга и удалённость от " +
			"крупных населённых пунктов создают практически идеальные условия для " +
			"астрономических наблюдений — световое загрязнение здесь минимально. " +
			"Широта 51.8° обеспечивает хорошие углы пролёта МКС. Континентальный климат " +
			"гарантирует ясные морозные ночи зимой, хотя летние наблюдения предпочтительнее " +
			"из-за комфортных температур. МКС видна здесь как яркая быстро движущаяся звезда " +
			"на фоне Млечного Пути, который отлично виден в безлунные ночи.",
	},
	{
		ID:                6,
		Name:              "Танегасима",
		Country:           "Япония",
		Latitude:          30.399,
		Longitude:         130.9707,
		Elevation:         20,
		Timezone:          "UTC+9 (JST)",
		BestTime:          "Декабрь — март",
		LightPollution:    "Низкая",
		WeatherConditions: "Субтропический океанический климат, мягкие зимы",
		ImageKey:          "tanegashima",
		VideoKey:          "tanegashima_video",
		Description: "Космический центр Танегасима (TNSC) — крупнейший космодром Японии, " +
			"управляемый агентством JAXA. Расположен на южной оконечности острова Танегасима " +
			"в префектуре Кагосима. Часто называется «самым красивым космодромом в мире» " +
			"благодаря живописному побережью и субтропической растительности. Широта 30° " +
			"обеспечивает благоприятную геометрию для наблюдения пролётов МКС. Океаническое " +
			"расположение острова минимизирует световое загрязнение, а мягкий субтропический " +
			"климат позволяет проводить наблюдения круглый год, хотя зимние месяцы " +
			"предпочтительнее из-за более прозрачной атмосферы. Наблюдения лучше проводить " +
			"с восточного побережья острова, где открытый горизонт над Тихим океаном " +
			"позволяет отслеживать МКС от момента появления до исчезновения.",
	},
}

var CurrentCalculation = Calculation{
	ID:              1,
	Title:           "Расчёт видимости МКС на 15.03.2026",
	ObservationDate: "15 марта 2026",
	Status:          "Завершён",
	CreatedAt:       "10 марта 2026",
	Moderator:       "Иванов А.С.",
	CalculationResult: "МКС наблюдаема из 3 из 3 выбранных точек в указанную дату. " +
		"Суммарное время видимости: 18 мин 12 сек. " +
		"Оптимальная точка наблюдения: Байконур (макс. высота 78°).",
	Services: []CalculationService{
		{
			ServiceID:        1,
			ObservationOrder: 1,
			IsPrimary:        true,
			ObserverName:     "Иванов Алексей Сергеевич",
			PositionResult:   "Высота 62°, азимут 215°, 19:42–19:48 МСК",
		},
		{
			ServiceID:        2,
			ObservationOrder: 2,
			IsPrimary:        false,
			ObserverName:     "Петрова Мария Ивановна",
			PositionResult:   "Высота 78°, азимут 187°, 20:15–20:21 +06",
		},
		{
			ServiceID:        5,
			ObservationOrder: 3,
			IsPrimary:        false,
			ObserverName:     "Сидоров Дмитрий Олегович",
			PositionResult:   "Высота 45°, азимут 302°, 02:05–02:10 +09",
		},
	},
}

func getServiceByID(id int) *Service {
	for i := range Services {
		if Services[i].ID == id {
			return &Services[i]
		}
	}
	return nil
}

func enrichService(s *Service) Service {
	enriched := *s
	enriched.ImageURL = MinioURL + "/" + s.ImageKey + ".jpg"
	enriched.VideoURL = MinioURL + "/" + s.VideoKey + ".mp4"
	return enriched
}
