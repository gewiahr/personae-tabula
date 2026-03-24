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
	"github.com/spf13/viper"
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
	cfg := loadConfig()

	db := initPostgres(cfg)
	defer db.Close()

	rdb := initRedis(cfg)
	defer rdb.Close()

	userRepo := postgres.NewUserRepository(db)
	tableRepo := postgres.NewTableRepository(db)
	eventRepo := postgres.NewEventRepository(db)
	roomCache := repoRedis.NewRoomCache(rdb)
	userCache := repoRedis.NewUserCache(rdb)

	userService := service.NewUserService(userRepo)
	tableService := service.NewTableService(tableRepo, eventRepo, roomCache)
	eventService := service.NewEventService(eventRepo, roomCache)

	hub := websocket.NewHub(eventService, tableService, roomCache, userCache)
	go hub.Run()

	wsHandler := websocket.NewWebSocketHandler(hub, userService, tableService)
	apiHandler := apiHttp.NewAPIHandler(userService, tableService, eventService)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /user/register", apiHttp.HttpWrapper(apiHandler.RegisterUser))
	mux.HandleFunc("GET /user/{id}", apiHttp.HttpWrapper(apiHandler.GetUser))

	mux.HandleFunc("POST /table", apiHttp.HttpWrapper(apiHandler.CreateTable))
	mux.HandleFunc("GET /tables", apiHttp.HttpWrapper(apiHandler.ListTables))
	mux.HandleFunc("GET /table/{id}", apiHttp.HttpWrapper(apiHandler.GetTable))
	mux.HandleFunc("GET /table/{id}/feed", apiHttp.HttpWrapper(apiHandler.GetTableFeed))
	//mux.HandleFunc("POST /table/{id}/events", apiHttp.HttpWrapper(apiHandler.CreateEvent))
	//mux.HandleFunc("DELETE /table/{id}", apiHandler.DeleteTable)

	mux.HandleFunc("GET /health", apiHttp.HttpWrapper(apiHandler.HealthCheck))

	mux.HandleFunc("GET /ws", wsHandler.HandleConnection)

	// mux.Handle("/", http.FileServer(http.Dir("./static")))

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      apiHttp.LoggingMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Server starting on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}

type Config struct {
	PostgresDSN string `json:"postgresDSN"`
	RedisAddr   string `json:"redisAddr"`
	RedisPass   string `json:"redisPass"`
	Port        string `json:"port"`
}

func loadConfig() *Config {

	config := new(Config)

	viper.SetConfigName("tabula")
	viper.SetConfigType("json")
	viper.AddConfigPath("./mnt")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %s", err)
	}

	if err := viper.Unmarshal(config); err != nil {
		log.Fatalf("Unable to decode into struct: %v", err)
	}

	return config
}

func initPostgres(cfg *Config) *bun.DB {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.PostgresDSN)))

	sqldb.SetMaxOpenConns(25)
	sqldb.SetMaxIdleConns(10)
	sqldb.SetConnMaxLifetime(5 * time.Minute)

	db := bun.NewDB(sqldb, pgdialect.New())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatal("Failed to connect to PostgreSQL:", err)
	}

	log.Println("Connected to PostgreSQL")

	if err := createTables(ctx, db); err != nil {
		log.Fatal("Failed to create tables:", err)
	}

	return db
}

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

func createTables(ctx context.Context, db *bun.DB) error {
	_, err := db.NewCreateTable().
		Model((*domain.User)(nil)).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateTable().
		Model((*domain.Table)(nil)).
		IfNotExists().
		ForeignKey(`(created_by) REFERENCES users(id) ON DELETE SET NULL`).
		Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateTable().
		Model((*domain.TableEvent)(nil)).
		IfNotExists().
		ForeignKey(`(table_id) REFERENCES tables(id) ON DELETE CASCADE`).
		ForeignKey(`(user_id) REFERENCES users(id) ON DELETE SET NULL`).
		Exec(ctx)
	if err != nil {
		return err
	}

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

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"time":   time.Now().String(),
	})
}
