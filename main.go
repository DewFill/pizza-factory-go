package main

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"pizza-factory-go/handlers"
	"pizza-factory-go/middleware"
	"pizza-factory-go/sqlc"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file")
	}

	// Set up database connection
	db, err := setupDatabase()
	if err != nil {
		log.Println("Error connecting to the database:", err)
		return
	}
	defer db.Close()

	queries := sqlc.New(db)

	router := http.NewServeMux()

	router.Handle("GET /orders/{order_id}", handlers.HandleGetOrder(ctx, queries))

	router.Handle("POST /orders", handlers.HandleCreateOrder(ctx, db, queries))

	router.Handle("POST /orders/{order_id}/items", handlers.HandleAddItemsToOrder(ctx, db, queries))

	router.Handle("POST /orders/{order_id}/done", middleware.AuthHeaderRequired(handlers.HandlerMakeOrderDone(ctx, db, queries)))
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

	router.Handle("GET /orders", middleware.AuthHeaderRequired(handlers.HandlerListOrders(ctx, db, queries)))
	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
	server := http.Server{
		Addr:    os.Getenv("APP_HOSTNAME") + ":" + os.Getenv("APP_PORT"),
		Handler: middleware.Logging(router),
	}

	log.Println("Listening on " + "http://" + os.Getenv("APP_HOSTNAME") + ":" + os.Getenv("APP_PORT"))
	err = server.ListenAndServe()
	if err != nil {
		log.Fatalf(err.Error())
	}
}
