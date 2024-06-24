package main

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"io"
	"log"
	"net/http"
	"os"
	"pizza-factory-go/middleware"
	"pizza-factory-go/response"
	"pizza-factory-go/sqlc"
)

type OrderWithItemIds struct {
	Id      string  `json:"order_id"`
	ItemIds []int32 `json:"items"`
	IsDone  bool    `json:"done"`
}

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

	router.HandleFunc("GET /orders/{order_id}", func(writer http.ResponseWriter, request *http.Request) {
		orderId := request.PathValue("order_id")
		order, err := queries.GetOrderWithItemIds(ctx, orderId)
		if err != nil {
			log.Println(err)
			response.WriteJSON(writer, "Order not found", 404)
			return
		}

		log.Println(orderId, order)
		response.WriteJSON(writer, order, 200)
	})

	router.HandleFunc("POST /orders", func(writer http.ResponseWriter, request *http.Request) {
		type RequestData struct {
			ItemIds []int32 `json:"item_ids"`
		}

		var data RequestData

		// Чтение и декодирование JSON из тела запроса
		decoder := json.NewDecoder(request.Body)
		decoder.DisallowUnknownFields()
		err := decoder.Decode(&data)
		if err != nil {
			response.WriteJSON(writer, "Error decoding JSON", http.StatusBadRequest)
			return
		}

		// create order
		order, err := CreateOrder(ctx, db, queries, data.ItemIds, false)
		if err != nil {
			log.Printf("Error creating order: %v", err)
			return
		}

		response.WriteJSON(writer, order, 200)
	})

	router.HandleFunc("POST /orders/{order_id}/items", func(writer http.ResponseWriter, request *http.Request) {
		// Читаем тело запроса
		orderId := request.PathValue("order_id")

		body, err := io.ReadAll(request.Body)
		if err != nil {
			http.Error(writer, "Failed to read request body", http.StatusInternalServerError)
			return
		}
		defer request.Body.Close()

		// Определяем срез для хранения данных
		var itemIds []int32

		// Декодируем JSON
		err = json.Unmarshal(body, &itemIds)
		if err != nil {
			http.Error(writer, "Failed to decode JSON", http.StatusBadRequest)
			return
		}

		err = AddItemsToOrder(ctx, db, queries, orderId, itemIds)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		writer.Write([]byte("Added items to order"))
	})

	orderChangeStatusHandler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		orderId := request.PathValue("order_id")
		tx, err := db.Begin()
		defer tx.Rollback()
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		qtx := queries.WithTx(tx)
		status, err := qtx.GetOrderStatus(ctx, orderId)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			log.Printf("Error setting order done: %v", err)
			return
		}

		if status == true {
			http.Error(writer, "order is already done", http.StatusInternalServerError)
			log.Println("order is already done")
			return
		}

		err = qtx.SetOrderDone(ctx, orderId)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			log.Printf("Error setting order done: %v", err)
			return
		}

		writer.Write([]byte("Order is done"))
		tx.Commit()
	})
	router.Handle("POST /orders/{order_id}/done", middleware.AuthHeaderRequired(orderChangeStatusHandler))

	ordersHandler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		doneQuery := request.URL.Query().Get("done")

		if doneQuery == "" {
			orders, err := queries.ListOrders(ctx)
			if err != nil {
				response.WriteJSON(writer, err, 200)
				return
			}

			response.WriteJSON(writer, orders, 200)
			return
		} else if doneQuery == "1" {
			orders, err := queries.ListOrdersByDone(ctx, true)
			if err != nil {
				response.WriteJSON(writer, err, 200)
				return
			}

			response.WriteJSON(writer, orders, 200)
			return
		} else if doneQuery == "0" {
			orders, err := queries.ListOrdersByDone(ctx, false)
			if err != nil {
				response.WriteJSON(writer, err, 200)
				return
			}

			response.WriteJSON(writer, orders, 200)
			return
		}

		response.WriteJSON(writer, "done query has invalid value", 200)
	})
	router.Handle("GET /orders", middleware.AuthHeaderRequired(ordersHandler))

	handler := middleware.Logging(router)

	server := http.Server{
		Addr:    os.Getenv("APP_HOSTNAME") + ":" + os.Getenv("APP_PORT"),
		Handler: handler,
	}

	log.Println("Listening on " + "http://" + os.Getenv("APP_HOSTNAME") + ":" + os.Getenv("APP_PORT"))
	err = server.ListenAndServe()
	if err != nil {
		log.Fatalf(err.Error())
	}
}

func CreateOrder(ctx context.Context, db *sql.DB, queries *sqlc.Queries, itemIds []int32, isDone bool) (OrderWithItemIds, error) {
	tx, err := db.Begin()
	if err != nil {
		return OrderWithItemIds{}, err
	}

	defer tx.Rollback()

	qtx := queries.WithTx(tx)
	order, _ := qtx.CreateOrder(ctx, isDone)

	var sliceItemIds []int32
	for _, id := range itemIds {
		var itemId int32
		itemId, err = qtx.CreateOrderItems(ctx, sqlc.CreateOrderItemsParams{OrderID: order.ID, ItemID: id})
		if err != nil {
			return OrderWithItemIds{}, err
		}
		sliceItemIds = append(sliceItemIds, itemId)
	}

	if err = tx.Commit(); err != nil {
		return OrderWithItemIds{}, err
	}

	return OrderWithItemIds{Id: order.ID, ItemIds: sliceItemIds, IsDone: order.IsDone}, nil
}

func AddItemsToOrder(ctx context.Context, db *sql.DB, queries *sqlc.Queries, orderId string, itemIds []int32) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	qtx := queries.WithTx(tx)
	order, err := qtx.GetOrder(ctx, orderId)
	if err != nil {
		return errors.New("order not found")
	}

	if order.Done == true {
		return errors.New("could not add new items to already done order")
	}

	for _, id := range itemIds {
		_, err = qtx.CreateOrderItems(ctx, sqlc.CreateOrderItemsParams{OrderID: order.OrderID, ItemID: id})
		if err != nil {
			return errors.New(fmt.Sprintf("could not add item with ID %d to order", id))
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}
