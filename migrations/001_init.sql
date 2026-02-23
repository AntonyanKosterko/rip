-- Миграция: создание таблиц для приложения «Видимость МКС»
-- 4 таблицы: users, observation_points (услуги), calculations (заявки), calculation_points (м-м)
-- 5 статусов заявок: черновик, удалён, сформирован, завершён, отклонён

-- Таблица пользователей
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(100) NOT NULL UNIQUE,
    full_name VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Таблица услуг (точки наблюдения МКС)
CREATE TABLE IF NOT EXISTS observation_points (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL UNIQUE,
    country VARCHAR(100) NOT NULL,
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    elevation INTEGER NOT NULL DEFAULT 0,
    timezone VARCHAR(50) NOT NULL,
    best_time VARCHAR(200),
    light_pollution VARCHAR(100),
    weather_conditions TEXT,
    description TEXT,
    image_url VARCHAR(500),
    video_url VARCHAR(500),
    status VARCHAR(20) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'deleted'))
);

-- Таблица заявок (расчёты видимости МКС)
-- Статусы: draft (черновик), deleted (удалён), formed (сформирован), completed (завершён), rejected (отклонён)
CREATE TABLE IF NOT EXISTS calculations (
    id SERIAL PRIMARY KEY,
    status VARCHAR(20) NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'deleted', 'formed', 'completed', 'rejected')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    creator_id INTEGER NOT NULL REFERENCES users(id),
    formed_at TIMESTAMP,
    completed_at TIMESTAMP,
    moderator_id INTEGER REFERENCES users(id),
    observation_date DATE,
    total_visibility VARCHAR(500)
);

-- Таблица м-м: точки наблюдения в заявке
-- Составной уникальный ключ (calculation_id, point_id)
CREATE TABLE IF NOT EXISTS calculation_points (
    id SERIAL PRIMARY KEY,
    calculation_id INTEGER NOT NULL REFERENCES calculations(id),
    point_id INTEGER NOT NULL REFERENCES observation_points(id),
    observation_order INTEGER NOT NULL DEFAULT 1,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    observer_name VARCHAR(255),
    position_result VARCHAR(500),
    UNIQUE (calculation_id, point_id)
);

-- Индексы
CREATE INDEX IF NOT EXISTS idx_calculations_status ON calculations(status);
CREATE INDEX IF NOT EXISTS idx_calculations_creator ON calculations(creator_id);
CREATE INDEX IF NOT EXISTS idx_calc_points_calc ON calculation_points(calculation_id);
CREATE INDEX IF NOT EXISTS idx_calc_points_point ON calculation_points(point_id);
CREATE INDEX IF NOT EXISTS idx_obs_points_status ON observation_points(status);

-- ============================================================
-- Начальные данные
-- ============================================================

-- Пользователи
INSERT INTO users (username, full_name, email) VALUES
    ('ivanov', 'Иванов Алексей Сергеевич', 'ivanov@bmstu.ru'),
    ('petrova', 'Петрова Мария Ивановна', 'petrova@bmstu.ru'),
    ('sidorov', 'Сидоров Дмитрий Олегович', 'sidorov@bmstu.ru')
ON CONFLICT (username) DO NOTHING;

-- Точки наблюдения (услуги)
INSERT INTO observation_points (name, country, latitude, longitude, elevation, timezone, best_time, light_pollution, weather_conditions, image_url, video_url, description, status) VALUES
(
    'Москва', 'Россия', 55.7558, 37.6173, 156, 'UTC+3 (МСК)',
    'Март — апрель, сентябрь — октябрь', 'Высокая',
    'Переменная облачность, лучшие условия в ясные зимние ночи',
    '/static/moscow.jpg',
    NULL,
    'Москва — столица Российской Федерации и один из крупнейших мегаполисов мира с населением более 13 миллионов человек. Несмотря на высокий уровень световой загрязнённости, МКС можно наблюдать невооружённым глазом благодаря её яркости (до −3.8 звёздной величины). Наилучшие условия наблюдения складываются в переходные сезоны — весной и осенью, когда ночи достаточно тёмные, а угол наклона орбиты МКС обеспечивает пролёт над средними широтами. Рекомендуется выбирать наблюдательные площадки на возвышенностях — Воробьёвы горы, смотровая площадка МГУ или парк «Коломенское» вдали от ярких источников света. Станция проходит по небу за 4–6 минут, двигаясь с запада на восток.',
    'active'
),
(
    'Байконур', 'Казахстан', 45.9647, 63.305, 90, 'UTC+6',
    'Круглый год, лучше всего летом', 'Низкая',
    'Преимущественно ясная погода, сухой континентальный климат',
    '/static/baikonur.jpg',
    NULL,
    'Космодром Байконур — первый и крупнейший в мире космодром, расположенный в Кызылординской области Казахстана. Именно отсюда 12 апреля 1961 года стартовал корабль «Восток-1» с Юрием Гагариным на борту. Благодаря удалённости от крупных городов и сухому климату степной зоны, здесь идеальные условия для наблюдения за МКС.',
    'active'
),
(
    'Мыс Канаверал', 'США', 28.3922, -80.6077, 3, 'UTC−5 (EST)',
    'Ноябрь — февраль', 'Средняя',
    'Субтропический климат, частая облачность летом',
    '/static/cape_canaveral.jpg',
    NULL,
    'Мыс Канаверал — легендарная стартовая площадка NASA на атлантическом побережье штата Флорида. Здесь расположены Космический центр Кеннеди и база ВВС США, откуда осуществлялись запуски программ «Аполлон», Space Shuttle и современных ракет SpaceX.',
    'active'
),
(
    'Куру', 'Французская Гвиана', 5.1594, -52.6503, 8, 'UTC−3',
    'Август — ноябрь (сухой сезон)', 'Низкая',
    'Экваториальный климат, высокая влажность, частые дожди',
    '/static/kourou.jpg',
    NULL,
    'Гвианский космический центр (Centre Spatial Guyanais) в городе Куру — основной космодром Европейского космического агентства (ESA). Расположен всего в 5 градусах от экватора.',
    'active'
),
(
    'Восточный', 'Россия', 51.8844, 128.334, 275, 'UTC+9',
    'Июнь — август', 'Очень низкая',
    'Континентальный климат, холодные зимы, тёплое лето',
    '/static/vostochny.jpg',
    NULL,
    'Космодром Восточный — новейший российский космодром, расположенный в Амурской области Дальнего Востока. Строительство началось в 2012 году, первый пуск состоялся 28 апреля 2016 года.',
    'active'
),
(
    'Танегасима', 'Япония', 30.399, 130.9707, 20, 'UTC+9 (JST)',
    'Декабрь — март', 'Низкая',
    'Субтропический океанический климат, мягкие зимы',
    '/static/tanegashima.jpg',
    NULL,
    'Космический центр Танегасима (TNSC) — крупнейший космодром Японии, управляемый агентством JAXA. Расположен на южной оконечности острова Танегасима в префектуре Кагосима.',
    'active'
)
ON CONFLICT (name) DO NOTHING;

-- Завершённая заявка (для демонстрации)
INSERT INTO calculations (status, creator_id, created_at, formed_at, completed_at, moderator_id, observation_date, total_visibility) VALUES
    ('completed', 1, '2026-03-10', '2026-03-12', '2026-03-14', 2, '2026-03-15',
     'МКС наблюдаема из 3 из 3 выбранных точек. Суммарное время видимости: 18 мин 12 сек. Оптимальная точка: Байконур (макс. высота 78°).')
ON CONFLICT DO NOTHING;

-- М-М для завершённой заявки
INSERT INTO calculation_points (calculation_id, point_id, observation_order, is_primary, observer_name, position_result) VALUES
    (1, 1, 1, TRUE, 'Иванов Алексей Сергеевич', 'Высота 62°, азимут 215°, 19:42–19:48 МСК'),
    (1, 2, 2, FALSE, 'Петрова Мария Ивановна', 'Высота 78°, азимут 187°, 20:15–20:21 +06'),
    (1, 5, 3, FALSE, 'Сидоров Дмитрий Олегович', 'Высота 45°, азимут 302°, 02:05–02:10 +09')
ON CONFLICT DO NOTHING;
