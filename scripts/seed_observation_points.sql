-- Наполнение observation_points для доп. задания «Индексы в БД» (>100 000 записей)
-- Запуск: psql или Adminer, один раз (занимает 1–3 мин)
-- postgres://root:root@localhost:5433/RIP

CREATE EXTENSION IF NOT EXISTS pg_trgm;

DO $$
DECLARE
  cnt BIGINT;
BEGIN
  SELECT COUNT(*) INTO cnt FROM observation_points WHERE status = 'active';
  IF cnt >= 100000 THEN
    RAISE NOTICE 'Уже % активных точек — пропуск seed', cnt;
    RETURN;
  END IF;

  INSERT INTO observation_points (
    name, country, latitude, longitude, elevation, timezone,
    best_time, light_pollution, weather_conditions, description, status
  )
  SELECT
    'Точка demo ' || LPAD(g::text, 6, '0'),
    CASE (g % 5)
      WHEN 0 THEN 'Россия'
      WHEN 1 THEN 'США'
      WHEN 2 THEN 'Казахстан'
      WHEN 3 THEN 'Япония'
      ELSE 'Франция'
    END,
    55.0 + (g % 100) * 0.01,
    37.0 + (g % 100) * 0.01,
    (g % 5000),
    'UTC+' || (g % 12),
    'Лето',
    'Низкая',
    'Ясно',
    'Демо-точка для лабораторной по индексам',
    'active'
  FROM generate_series(1, 120000) AS g;

  RAISE NOTICE 'Добавлено 120000 точек';
END $$;

ANALYZE observation_points;

SELECT COUNT(*) AS active_points FROM observation_points WHERE status = 'active';
