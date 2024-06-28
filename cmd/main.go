package main

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"pizza-factory-go/handlers"
	"pizza-factory-go/middleware"
	"pizza-factory-go/service"
	"pizza-factory-go/sqlc"
)

func main() {
	// Set up database connection
	db, err := setupDatabase()
	if err != nil {
		log.Println("Error connecting to the database:", err)
		return
	}
	defer db.Close()

	// Create context and slqc queries
	ctx := context.Background()
	queries := sqlc.New(db)

	// Create the order service
	orderService := service.NewOrderService(db, queries)

	// Set up router
	router := http.NewServeMux()

	// Register handlers
	router.Handle("POST /orders", handlers.HandlerCreateOrder(ctx, orderService))
	router.Handle("POST /orders/{order_id}/items", handlers.HandlerAddItemsToOrder(ctx, orderService))
	router.Handle("GET /orders/{order_id}", handlers.HandlerGetOrder(ctx, orderService))
	router.Handle("POST /orders/{order_id}/done", middleware.AuthHeaderRequired(handlers.HandlerMakeOrderDone(ctx, orderService)))
	router.Handle("GET /orders", middleware.AuthHeaderRequired(handlers.HandlerListOrders(ctx, orderService)))

	// Start the server
	startServer(router)
}

// setupDatabase initializes the database connection using environment variables
func setupDatabase() (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

// startServer starts the HTTP server with logging and address from environment variables
func startServer(router *http.ServeMux) {
	server := http.Server{
		Addr:    fmt.Sprintf("%s:%s", os.Getenv("APP_HOSTNAME"), os.Getenv("APP_PORT")),
		Handler: middleware.Logging(router),
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}

	log.Printf("Listening on http://%s:%s", os.Getenv("APP_HOSTNAME"), os.Getenv("APP_PORT"))
}
