package main

import (
	"log"
	"net/http"
	"os"
	"task11/calendar"
	"task11/httpserver"
)

func main() {
	// создаём хранилище
	store := calendar.NewStore()

	// создаём сервер
	server := httpserver.NewServer(store)

	// создаём mux (роутер)
	mux := http.NewServeMux()

	// регистрируем маршруты
	server.RegisterRoutes(mux)

	// оборачиваем в middleware
	loggedMux := httpserver.LoggingMiddleware(mux)

	// порт
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Println("Server started on :" + port)

	// запуск
	err := http.ListenAndServe(":"+port, loggedMux)
	if err != nil {
		log.Fatal(err)
	}
}
