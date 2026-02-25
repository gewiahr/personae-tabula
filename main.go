package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"personae-tabula/internal/domain"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"

	apiHttp "personae-tabula/internal/api/http"
	"personae-tabula/internal/repository/postgres"
	repoRedis "personae-tabula/internal/repository/redis"
	"personae-tabula/internal/service"
	"personae-tabula/internal/websocket"
)

func main() {
	// Загружаем конфигурацию
	cfg := loadConfig()

	// Инициализируем базы данных
	db := initPostgres(cfg)
	defer db.Close()

	rdb := initRedis(cfg)
	defer rdb.Close()

	// Инициализируем репозитории
	userRepo := postgres.NewUserRepository(db)
	tableRepo := postgres.NewTableRepository(db)
	eventRepo := postgres.NewEventRepository(db)
	roomCache := repoRedis.NewRoomCache(rdb)

	// Инициализируем сервисы
	userService := service.NewUserService(userRepo)
	tableService := service.NewTableService(tableRepo, eventRepo, roomCache)
	eventService := service.NewEventService(eventRepo, roomCache)

	// Инициализируем WebSocket хаб
	hub := websocket.NewHub(eventService, tableService, roomCache)
	go hub.Run()

	// Инициализируем обработчики
	wsHandler := websocket.NewWebSocketHandler(hub, userService, tableService)
	apiHandler := apiHttp.NewAPIHandler(userService, tableService, eventService)

	// Настраиваем роутинг
	mux := http.NewServeMux()

	// API эндпоинты (REST)
	mux.HandleFunc("POST /api/users", apiHandler.CreateUser)
	mux.HandleFunc("GET /api/users/{id}", apiHandler.GetUser)

	mux.HandleFunc("POST /api/tables", apiHandler.CreateTable)
	mux.HandleFunc("GET /api/tables", apiHandler.ListTables)
	mux.HandleFunc("GET /api/tables/{id}", apiHandler.GetTable)
	mux.HandleFunc("GET /api/tables/{id}/feed", apiHandler.GetTableFeed)
	mux.HandleFunc("POST /api/tables/{id}/events", apiHandler.CreateEvent)
	//mux.HandleFunc("DELETE /api/tables/{id}", apiHandler.DeleteTable)

	mux.HandleFunc("GET /api/health", apiHandler.HealthCheck)

	// WebSocket
	mux.HandleFunc("GET /ws", wsHandler.HandleConnection)

	// Статика для тестов (если нужна)
	mux.Handle("/", http.FileServer(http.Dir("./static")))

	// Создаем сервер с таймаутами
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		log.Printf("Server starting on port %d", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Ждем сигнал завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Даем время завершить текущие соединения
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}

// Конфигурация
type Config struct {
	PostgresDSN string
	RedisAddr   string
	RedisPass   string
	Port        int
}

func loadConfig() *Config {
	// Можно загружать из .env файла или переменных окружения
	return &Config{
		PostgresDSN: os.Getenv("DATABASE_URL"),
		RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPass:   os.Getenv("REDIS_PASSWORD"),
		Port:        getEnvAsInt("PORT", 8080),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intVal int
		fmt.Sscanf(value, "%d", &intVal)
		return intVal
	}
	return defaultValue
}

// PostgreSQL инициализация с bun
func initPostgres(cfg *Config) *bun.DB {
	// Для PostgreSQL
	dsn := cfg.PostgresDSN
	if dsn == "" {
		dsn = "postgres://postgres:password@localhost:5432/dice?sslmode=disable"
	}

	// Создаем подключение
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))

	// Настраиваем пул соединений
	sqldb.SetMaxOpenConns(25)
	sqldb.SetMaxIdleConns(10)
	sqldb.SetConnMaxLifetime(5 * time.Minute)

	// Создаем bun.DB
	db := bun.NewDB(sqldb, pgdialect.New())

	// Проверяем подключение
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatal("Failed to connect to PostgreSQL:", err)
	}

	log.Println("Connected to PostgreSQL")

	// Создаем таблицы (только для разработки)
	if err := createTables(ctx, db); err != nil {
		log.Fatal("Failed to create tables:", err)
	}

	return db
}

// Redis инициализация
func initRedis(cfg *Config) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPass,
		DB:       0,
		PoolSize: 10,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}

	log.Println("Connected to Redis")
	return client
}

// Создание таблиц (миграции)
func createTables(ctx context.Context, db *bun.DB) error {
	// Пользователи
	_, err := db.NewCreateTable().
		Model((*domain.User)(nil)).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return err
	}

	// Столы
	_, err = db.NewCreateTable().
		Model((*domain.Table)(nil)).
		IfNotExists().
		ForeignKey(`(created_by) REFERENCES users(id) ON DELETE SET NULL`).
		Exec(ctx)
	if err != nil {
		return err
	}

	// События
	_, err = db.NewCreateTable().
		Model((*domain.TableEvent)(nil)).
		IfNotExists().
		ForeignKey(`(table_id) REFERENCES tables(id) ON DELETE CASCADE`).
		ForeignKey(`(user_id) REFERENCES users(id) ON DELETE SET NULL`).
		Exec(ctx)
	if err != nil {
		return err
	}

	// Индексы для производительности
	_, err = db.NewCreateIndex().
		Model((*domain.TableEvent)(nil)).
		Index("idx_table_events_table_id").
		IfNotExists().
		Column("table_id").
		Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().
		Model((*domain.TableEvent)(nil)).
		Index("idx_table_events_created_at").
		IfNotExists().
		Column("created_at").
		Exec(ctx)
	if err != nil {
		return err
	}

	log.Println("Database tables created/verified")
	return nil
}

// Middleware для логирования
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Логируем запрос
		log.Printf("[%s] %s %s", r.Method, r.URL.Path, time.Since(start))

		// Добавляем заголовки CORS для разработки
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Health check
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"time":   time.Now().String(),
	})
}
