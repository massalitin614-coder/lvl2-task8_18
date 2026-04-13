package main

import (
	"fmt"
	"os"
	"time"

	"github.com/beevik/ntp"
)

func getNTP(server string) (time.Time, error) {
	return ntp.Time(server)
}

// Возращаем специальную структуру
func getNTPTime(server string) (*ntp.Response, error) {
	return ntp.Query(server)
}

func main() {
	server := "0.beevik-ntp.pool.ntp.org"
	//1. Просто выводим время
	resultNTP, err := getNTP(server)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get NTP server")
		os.Exit(1)
	}

	fmt.Printf("Точное время текущего NTP: %v\n", resultNTP.Format(time.RFC1123))
	fmt.Println("==================================================")
	//2. Выводим структуру query
	response, err := getNTPTime(server)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to query NTP server: %v\n", err)
		os.Exit(1)
	}
	//Проверяем сервер на валидность
	if err = response.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "The server response is defective: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Анализ NTP-сервера : %s\n", server)

	fmt.Printf("Время на сервере (по его мнению): %v\n", response.Time)

	fmt.Printf("Смещение твоих часов: %v\n", response.ClockOffset)

	fmt.Printf("Задержка сети (RTT): %v\n", response.RTT)

	fmt.Printf("Уровень точности сервера (Stratum): %d\n", response.Stratum)

	preciseTime := time.Now().Add(response.ClockOffset)

	fmt.Printf("Точное время текущего NTP: %v\n", preciseTime.Format(time.RFC3339))
}
