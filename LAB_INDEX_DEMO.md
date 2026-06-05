# Доп. задание: Индексы в БД (+4 балла)

## Подготовка данных (один раз)

```bash
psql "postgres://root:root@localhost:5433/RIP" -f scripts/seed_observation_points.sql
```

Проверка: `SELECT COUNT(*) FROM observation_points WHERE status = 'active';` — должно быть **> 100 000**.

## API

```
GET /api/services?search=рос&page=1&page_size=20
```

Ответ:

```json
{
  "items": [...],
  "total": 24000,
  "page": 1,
  "page_size": 20
}
```

При `viewed=1` — по-прежнему массив (страница «Подробнее»).

## Демонстрация индекса на защите

1. Открыть React → список услуг → поиск «рос» → перелистать страницы (Network: `page`, `page_size`, `search`).
2. В Adminer/psql выполнить [scripts/lab_index_demo.sql](scripts/lab_index_demo.sql) по шагам:
   - `DROP INDEX` → `EXPLAIN ANALYZE` → **Seq Scan**, записать `Execution Time`.
   - `CREATE INDEX idx_obs_points_search_trgm` → `ANALYZE` → тот же `EXPLAIN ANALYZE` → **Bitmap Index Scan** / GIN, время меньше.
3. Скриншоты обоих планов — в РПЗ (Задачи и Заключение).

Индекс (GIN + pg_trgm) для `LIKE '%...%'`:

```sql
CREATE INDEX idx_obs_points_search_trgm
  ON observation_points
  USING gin ((lower(name) || ' ' || lower(country)) gin_trgm_ops)
  WHERE status = 'active';
```
