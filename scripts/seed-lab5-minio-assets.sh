#!/usr/bin/env bash
# Кладёт в iss-bucket объекты lab5/* — те же пути, что в frontend/src/mock/services.ts
# Запуск из корня репозитория: bash scripts/seed-lab5-minio-assets.sh

set -euo pipefail
cd "$(dirname "$0")/.."

if ! docker inspect iss-minio &>/dev/null; then
  echo "Контейнер iss-minio не найден. Запусти: docker compose up -d minio minio-init"
  exit 1
fi

NET=$(docker inspect iss-minio --format '{{range $k, $v := .NetworkSettings.Networks}}{{$k}}{{"\n"}}{{end}}' | head -n1)
echo "Сеть Docker: $NET"

TMP=$(mktemp -d)
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

dl() {
  local url="$1" out="$2"
  curl -fsSL --connect-timeout 30 -A "Mozilla/5.0 (compatible; RIP-lab5-seed/1.0)" "$url" -o "$out" 2>/dev/null || return 1
}

echo "Скачивание файлов (picsum — стабильный seed, без 429 Wikimedia)..."
# Детерминированные JPEG по seed (picsum.photos)
dl "https://picsum.photos/seed/rip-baikonur/960/640.jpg" "$TMP/baikonur.jpg"
dl "https://picsum.photos/seed/rip-moscow/960/640.jpg" "$TMP/moscow.jpg"
dl "https://picsum.photos/seed/rip-arkhiz/960/640.jpg" "$TMP/arkhiz.jpg"
dl "https://picsum.photos/seed/rip-teide/960/640.jpg" "$TMP/teide.jpg"
dl "https://picsum.photos/seed/rip-atacama/960/640.jpg" "$TMP/atacama.jpg"
dl "https://picsum.photos/seed/rip-mauna/960/640.jpg" "$TMP/mauna.jpg"
dl "https://picsum.photos/seed/rip-simeiz/960/640.jpg" "$TMP/simeiz.jpg"
dl "https://picsum.photos/seed/rip-karaganda/960/640.jpg" "$TMP/karaganda.jpg"

for f in baikonur moscow arkhiz teide atacama mauna simeiz karaganda; do
  [[ -s "$TMP/$f.jpg" ]] || { echo "Пустой файл: $f.jpg"; exit 1; }
done

echo "Скачивание демо-видео (mp4)..."
dl "https://filesamples.com/samples/video/mp4/sample_640x360.mp4" "$TMP/baikonur.mp4" || \
  dl "http://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerBlazes.mp4" "$TMP/baikonur.mp4"
[[ -s "$TMP/baikonur.mp4" ]] || { echo "Не удалось скачать mp4."; exit 1; }
cp "$TMP/baikonur.mp4" "$TMP/moscow.mp4"

echo "Загрузка в MinIO (mc)..."
docker run --rm \
  --network "$NET" \
  -v "$TMP:/seed:ro" \
  minio/mc:latest \
  sh -ce "
mc alias set local http://minio:9000 minioadmin minioadmin
mc mb -p local/iss-bucket || true
mc anonymous set download local/iss-bucket || true
mc cp /seed/baikonur.jpg local/iss-bucket/lab5/baikonur.jpg
mc cp /seed/moscow.jpg local/iss-bucket/lab5/moscow.jpg
mc cp /seed/arkhiz.jpg local/iss-bucket/lab5/arkhiz.jpg
mc cp /seed/teide.jpg local/iss-bucket/lab5/teide.jpg
mc cp /seed/atacama.jpg local/iss-bucket/lab5/atacama.jpg
mc cp /seed/mauna.jpg local/iss-bucket/lab5/mauna.jpg
mc cp /seed/simeiz.jpg local/iss-bucket/lab5/simeiz.jpg
mc cp /seed/karaganda.jpg local/iss-bucket/lab5/karaganda.jpg
mc cp /seed/baikonur.mp4 local/iss-bucket/lab5/baikonur.mp4
mc cp /seed/moscow.mp4 local/iss-bucket/lab5/moscow.mp4
"

echo "Готово. Проверка:"
curl -sI "http://localhost:9000/iss-bucket/lab5/baikonur.jpg" | head -3 || true
