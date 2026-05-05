-- ЛР6: демонстрационные SQL-запросы для показа "до/после" во frontend.
-- Запускать в Adminer или psql к БД RIP.

-- 1) Проверка исходного состояния данных каталога.
SELECT
  id,
  name,
  country,
  timezone,
  elevation,
  image_url
FROM observation_points
WHERE status = 'active'
ORDER BY id;

-- 2) Изменение карточки, которое сразу видно в списке/детали во frontend.
-- Подставьте id существующей услуги (пример: 1).
UPDATE observation_points
SET
  name = 'Москва (ЛР6 demo)',
  timezone = 'UTC+3',
  elevation = 180,
  description = 'CLIP demo point in Moscow with clear night observations for ISS.',
  image_url = 'http://localhost:9000/iss-bucket/lab6_demo_moscow.jpg'
WHERE id = 1;

-- 3) Выборка после изменения (подтверждение в БД).
SELECT
  id,
  name,
  timezone,
  elevation,
  image_url,
  description
FROM observation_points
WHERE id = 1;

-- 4) Проверка MinIO URL, которые фронтенд должен получить через API.
SELECT
  id,
  name,
  image_url,
  split_part(image_url, '/', array_length(string_to_array(image_url, '/'), 1)) AS image_object_name
FROM observation_points
WHERE status = 'active'
ORDER BY id;

-- 5) Откат демонстрационного изменения (после защиты, по желанию).
-- UPDATE observation_points
-- SET
--   name = 'Москва',
--   timezone = 'UTC+3 (МСК)',
--   elevation = 156,
--   description = 'Москва — столица Российской Федерации и один из крупнейших мегаполисов мира с населением более 13 миллионов человек. Несмотря на высокий уровень световой загрязнённости, МКС можно наблюдать невооружённым глазом благодаря её яркости (до −3.8 звёздной величины). Наилучшие условия наблюдения складываются в переходные сезоны — весной и осенью, когда ночи достаточно тёмные, а угол наклона орбиты МКС обеспечивает пролёт над средними широтами. Рекомендуется выбирать наблюдательные площадки на возвышенностях — Воробьёвы горы, смотровая площадка МГУ или парк «Коломенское» вдали от ярких источников света. Станция проходит по небу за 4–6 минут, двигаясь с запада на восток.',
--   image_url = '/static/moscow.jpg'
-- WHERE id = 1;
