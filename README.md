## WB L0 — сервис заказов

Небольшой сервис на Go: читает JSON‑заказы из Kafka, сохраняет их в Postgres, кэширует последние заказы в памяти и отдает по HTTP вместе с простой статической страницей.

### Возможности
- **Потребитель Kafka**: читает из топика `orders` (опционально DLQ `orders_dlq`).
- **Хранение в Postgres**: нормализованная схема с таблицами `orders`, `deliveries`, `payments`, `items`.
- **Памятный кэш**: «горячие» заказы лежат в памяти для быстрых чтений.
- **HTTP API + статика**: `GET /order/{uid}` и простая страница из каталога `web/`.
- **Миграции**: через `golang-migrate` (в Docker Compose запускаются автоматически).

### Технологии
- Go 1.24
- Kafka (образы Confluent), Zookeeper
- Postgres 15
- chi, segmentio/kafka-go

---

## Быстрый старт (Docker Compose)

Требования: Docker Desktop/Engine и Docker Compose.

1) Запустите стек:
```bash
make up
```

Будут подняты:
- Zookeeper (`2181`)
- Kafka (`9092` наружу, `29092` внутри сети)
- Postgres (`5432`)
- Adminer (`http://localhost:8081`)
- Приложение (`http://localhost:8080`)

Миграции применяются контейнером `migrate` автоматически.

2) Откройте веб‑страницу: `http://localhost:8080`

3) Отправьте пример заказа в Kafka (с хоста):
```bash
go run ./cmd/producer
```

4) Получите заказ по UID (подставьте реальный `order_uid` после попадания в БД):
```bash
curl http://localhost:8080/order/<order_uid>
```

Остановка стека:
```bash
make down
```

Просмотр логов:
```bash
make logs
```

---

## Локальный запуск (без Docker)

Требования:
- Go 1.24+
- Локальный Postgres 15+
- Локальный брокер Kafka на `localhost:9092`
- CLI `golang-migrate` для миграций

1) Настройте Postgres (пример переменных):
```
POSTGRES_USER=wb
POSTGRES_PASSWORD=wb
POSTGRES_DB=wb_orders
```

2) Примените миграции:
```bash
make migrate-up
```

3) Запустите приложение:
```bash
make run
```

4) Отправьте пример сообщения:
```bash
go run ./cmd/producer
```

Переменные окружения (значения по умолчанию):
```
PG_HOST=localhost
PG_PORT=5432
PG_USER=wb
PG_PASSWORD=wb
PG_DB=wb_orders
KAFKA_BROKER=localhost:9092
KAFKA_TOPIC=orders
KAFKA_GROUP=orders-consumer
HTTP_ADDR=:8080
```

Можно создать файл `.env` в корне проекта — приложение загрузит его при старте.

---

## API

- GET `/order/{uid}` → возвращает один заказ в JSON. Сначала кэш, затем БД.

Пример:
```bash
curl http://localhost:8080/order/b563feb7b2b84b6test
```

Ответ (схема укорочена):
```json
{
   "order_uid": "b563feb7b2b84b6test",
   "track_number": "WBILMTESTTRACK",
   "entry": "WBIL",
   "delivery": {
      "name": "Test Testov",
      "phone": "+9720000000",
      "zip": "2639809",
      "city": "Kiryat Mozkin",
      "address": "Ploshad Mira 15",
      "region": "Kraiot",
      "email": "test@gmail.com"
   },
   "payment": {
      "transaction": "b563feb7b2b84b6test",
      "request_id": "",
      "currency": "USD",
      "provider": "wbpay",
      "amount": 1817,
      "payment_dt": 1637907727,
      "bank": "alpha",
      "delivery_cost": 1500,
      "goods_total": 317,
      "custom_fee": 0
   },
   "items": [
      {
         "chrt_id": 9934930,
         "track_number": "WBILMTESTTRACK",
         "price": 453,
         "rid": "ab4219087a764ae0btest",
         "name": "Mascaras",
         "sale": 30,
         "size": "0",
         "total_price": 317,
         "nm_id": 2389212,
         "brand": "Vivienne Sabo",
         "status": 202
      }
   ],
   "locale": "en",
   "internal_signature": "",
   "customer_id": "test",
   "delivery_service": "meest",
   "shardkey": "9",
   "sm_id": 99,
   "date_created": "2021-11-26T06:22:19Z",
   "oof_shard": "1"
}
```

Статика из каталога `web/` доступна по корню `/`.

---

## Kafka

- Конфигурация потребителя задаётся в `cmd/app/main.go` через переменные окружения:
  - `KAFKA_BROKER` (например, `kafka:9092` в Docker или `localhost:9092` локально)
  - `KAFKA_TOPIC` по умолчанию `orders`
  - `KAFKA_GROUP` по умолчанию `orders-consumer`
- Опциональный DLQ (`orders_dlq`) включён в коде. Некорректный JSON или неверные бизнес‑данные отправляются в DLQ при его наличии; иначе — пропускаются с логированием.

Продюсер‑пример: `cmd/producer/main.go` читает `model.json` и отправляет в топик `orders` на `localhost:9092`.

---

## База данных

Строка подключения: `postgres://<user>:<pass>@<host>:<port>/<db>?sslmode=disable`.

Миграции находятся в `migrations/` и применяются командами:
```bash
make migrate-up      # применить все
make migrate-down    # откатить одну
make migrate-version # показать текущую версию
```

Adminer доступен по адресу `http://localhost:8081` (server: `postgres`, user: `wb`, pass: `wb`, db: `wb_orders`).

---

## Цели Makefile

```make
run               # go run ./cmd/app
build             # go build -o bin/wb-orders ./cmd/app
tidy              # go mod tidy
up                # docker compose up -d
down              # docker compose down
logs              # docker compose logs -f
migrate-up        # применить миграции (нужен golang-migrate)
migrate-down      # откатить одну миграцию
migrate-force     # выставить версию=1 (при необходимости)
migrate-version   # показать версию миграций
```

---

## Структура проекта

```
cmd/app             # точка входа сервиса (HTTP + Kafka consumer)
cmd/producer        # небольшой продюсер для примера
internal/httpapi    # роутер chi, обработчики, раздача статики
internal/kafka      # потребитель kafka-go с ручными коммитами и DLQ
internal/storage    # доступ к Postgres (insert, get, прогрев)
internal/cache      # кэш в памяти
internal/models     # модель заказа
migrations          # SQL-миграции
web                 # статические файлы
```

---

## Советы по разработке
- Используйте `.env` для локальных запусков.
- При изменении схемы добавляйте новые миграции, не редактируйте старые.
- Дубликаты заказов не страшны: вставка идемпотентна по `order_uid`, позиции обновляются.

---

## Диагностика проблем
- Ошибки подключения к Kafka: проверьте доступность брокера (`localhost:9092` локально или `kafka:29092` внутри сети Docker).
- Нет соединения с Postgres: проверьте здоровье контейнера и проброс порта `5432`.
- Миграции не применились: выполните `make migrate-up` локально или перезапустите контейнер `migrate`.
- 404 на `/order/{uid}`: убедитесь, что UID существует; сначала опубликуйте `model.json` продюсером.

---
