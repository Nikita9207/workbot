package main

import (
	"fmt"
	"log"

	"workbot/internal/gsheets"
)

func main() {
	// Тест OAuth2 клиента
	client, err := gsheets.NewOAuthClient(
		"oauth-credentials.json",
		"google-token.json",
		"", // без папки - создаст в корне Drive
	)
	if err != nil {
		log.Fatalf("Ошибка инициализации: %v", err)
	}

	fmt.Println("Google Sheets клиент инициализирован!")

	// Создаём тестовую таблицу
	sheetID, err := client.CreateClientSpreadsheet(999, "Тест", "Клиент")
	if err != nil {
		log.Fatalf("Ошибка создания таблицы: %v", err)
	}

	fmt.Printf("Таблица создана: https://docs.google.com/spreadsheets/d/%s\n", sheetID)
}
