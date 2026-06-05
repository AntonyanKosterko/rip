-- Демонстрация индекса для текстового search (как GET /api/services?search=...)
-- Запускать в Adminer/psql по шагам, сравнивать Execution Time

CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Шаг 1: без индекса (или после DROP)
DROP INDEX IF EXISTS idx_obs_points_search_trgm;

EXPLAIN ANALYZE
SELECT id, name, country, elevation
FROM observation_points
WHERE status = 'active'
  AND (lower(name) || ' ' || lower(country)) LIKE '%рос%'
ORDER BY id
LIMIT 20 OFFSET 0;

-- Шаг 2: создать GIN-индекс под LIKE '%...%'
CREATE INDEX idx_obs_points_search_trgm
  ON observation_points
  USING gin ((lower(name) || ' ' || lower(country)) gin_trgm_ops)
  WHERE status = 'active';

ANALYZE observation_points;

EXPLAIN ANALYZE
SELECT id, name, country, elevation
FROM observation_points
WHERE status = 'active'
  AND (lower(name) || ' ' || lower(country)) LIKE '%рос%'
ORDER BY id
LIMIT 20 OFFSET 0;
