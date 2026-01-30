package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
)

func main() {
	// Читаем OAuth credentials
	credFile := "oauth-credentials.json"
	if len(os.Args) > 1 {
		credFile = os.Args[1]
	}

	b, err := os.ReadFile(credFile)
	if err != nil {
		log.Fatalf("Не удалось прочитать %s: %v", credFile, err)
	}

	config, err := google.ConfigFromJSON(b,
		sheets.SpreadsheetsScope,
		"https://www.googleapis.com/auth/drive",
	)
	if err != nil {
		log.Fatalf("Ошибка парсинга credentials: %v", err)
	}

	// Используем localhost для получения кода
	config.RedirectURL = "http://localhost:8090/callback"

	// Канал для получения кода
	codeCh := make(chan string)

	// Запускаем локальный сервер для получения callback
	srv := &http.Server{Addr: ":8090"}
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			fmt.Fprintf(w, "Ошибка: код не получен")
			return
		}
		fmt.Fprintf(w, "<h1>Авторизация успешна!</h1><p>Можете закрыть эту вкладку и вернуться в терминал.</p>")
		codeCh <- code
	})

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("HTTP сервер: %v", err)
		}
	}()

	// Генерируем URL для авторизации
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	fmt.Printf("\n=== Google OAuth2 Setup ===\n\n")
	fmt.Printf("Откройте эту ссылку в браузере:\n\n%s\n\n", authURL)
	fmt.Println("Ожидаю авторизации...")

	// Ждём код
	code := <-codeCh

	// Останавливаем сервер
	srv.Shutdown(context.Background())

	// Обмениваем код на токен
	token, err := config.Exchange(context.Background(), code)
	if err != nil {
		log.Fatalf("Ошибка обмена кода на токен: %v", err)
	}

	fmt.Printf("\n=== Токен получен! ===\n\n")
	fmt.Printf("Access Token:  %s...\n", token.AccessToken[:20])
	fmt.Printf("Refresh Token: %s\n", token.RefreshToken)
	fmt.Printf("Token Type:    %s\n", token.TokenType)
	fmt.Printf("Expiry:        %s\n\n", token.Expiry)

	// Сохраняем токен в файл
	tokenFile := "google-token.json"
	f, err := os.Create(tokenFile)
	if err != nil {
		log.Fatalf("Ошибка создания файла токена: %v", err)
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(token); err != nil {
		log.Fatalf("Ошибка записи токена: %v", err)
	}

	fmt.Printf("Токен сохранён в: %s\n", tokenFile)
	fmt.Println("\nДобавьте в .env:")
	fmt.Printf("GOOGLE_OAUTH_CREDENTIALS_PATH=oauth-credentials.json\n")
	fmt.Printf("GOOGLE_TOKEN_PATH=google-token.json\n")
}
