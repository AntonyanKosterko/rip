-- ЛР ДЗ: индексы — расширение для текстового поиска (GIN pg_trgm)
-- Наполнение 120k+ строк: scripts/seed_observation_points.sql (вручную, один раз)
-- Демо-индекс: scripts/lab_index_demo.sql

CREATE EXTENSION IF NOT EXISTS pg_trgm;
