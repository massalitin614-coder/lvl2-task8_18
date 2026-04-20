# 📅 Go Calendar HTTP Server

Простой HTTP-сервер календаря на Go с CRUD операциями и фильтрацией событий.

## 🚀 Возможности

* Создание событий
* Обновление событий
* Удаление событий
* Получение событий:

  * за день
  * за неделю
  * за месяц
* Middleware логирования
* Потокобезопасное хранилище (mutex)
* Unit-тесты

---

## 🛠️ Технологии

* Go (net/http)
* In-memory storage
* JSON API

---

## ▶️ Запуск

```bash
go run main.go
```

Сервер стартует на:

```
http://localhost:8081
```

---

## 📡 API

### ➕ Создать событие

```
POST /create_event
```

Пример:

```bash
curl -X POST http://localhost:8081/create_event \
  -d "user_id=1&date=2026-04-19&event=test"
```

---

### 📅 События за день

```
GET /events_for_day?user_id=1&date=2026-04-19
```

---

### 📅 События за неделю

```
GET /events_for_week?user_id=1&date=2026-04-19
```

---

### 📅 События за месяц

```
GET /events_for_month?user_id=1&date=2026-04-19
```

---

### ❌ Удалить событие

```
POST /delete_event
```

```bash
curl -X POST http://localhost:8081/delete_event -d "id=1"
```

---

## 🧪 Тесты

```bash
go test -v ./...
```

Проверка на race condition:

```bash
go test -race ./...
```

---

## 🧠 Архитектура

```
main.go
  ↓
httpserver (handlers + middleware)
  ↓
calendar (business logic)
```

---

## 📌 Особенности

* Чистое разделение слоёв
* Нет зависимости бизнес-логики от HTTP
* Потокобезопасность
* Простая расширяемость

---