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
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file")
	}

	ctx := context.Background()

	db, err := sql.Open("postgres", "postgres://postgres:password@localhost/postgres?sslmode=disable")
	if err != nil {
		fmt.Println("Error connecting to the database:", err)
		return
	}
	defer db.Close()

	queries := sqlc.New(db)

	router := http.NewServeMux()

	router.Handle("GET /orders/{order_id}", handlers.HandleGetOrder(ctx, queries))

	router.Handle("POST /orders", handlers.HandleCreateOrder(ctx, db, queries))

	router.Handle("POST /orders/{order_id}/items", handlers.HandleAddItemsToOrder(ctx, db, queries))

	router.Handle("POST /orders/{order_id}/done", middleware.AuthHeaderRequired(handlers.HandlerMakeOrderDone(ctx, db, queries)))

	router.Handle("GET /orders", middleware.AuthHeaderRequired(handlers.HandlerListOrders(ctx, db, queries)))

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
