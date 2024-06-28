package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/lib/pq"
	"io"
	"log"
	"net/http"
	"pizza-factory-go/custom_errors"
	"pizza-factory-go/response"
	"pizza-factory-go/sqlc"
	"strconv"
)

// HandlerGetOrder handles the GET request to retrieve an order by its ID
func HandlerGetOrder(ctx context.Context, queries *sqlc.Queries) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		orderId := request.PathValue("order_id")
		order, err := queries.GetOrderWithItemIds(ctx, orderId)
		if err != nil {
			response.WritePlainText(writer, "order not found", http.StatusNotFound)
			return
		}

		response.WriteJSON(writer, order, http.StatusOK)
	})
}

// HandlerCreateOrder handles the POST request to create a new order
func HandlerCreateOrder(ctx context.Context, db *sql.DB, queries *sqlc.Queries) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		type RequestData struct {
			ItemIds []int32 `json:"items"`
		}

		var data RequestData

		// decoding and reading JSON from the request body
		decoder := json.NewDecoder(request.Body)
		decoder.DisallowUnknownFields()
		err := decoder.Decode(&data)
		if err != nil {
			response.WritePlainText(writer, "error decoding JSON", http.StatusBadRequest)
			return
		}

		// creating new order
		order, err := createOrder(ctx, db, queries, data.ItemIds, false)
		if err != nil {
			if errors.As(err, &custom_errors.ErrIdDoesNotExist) {
				response.WritePlainText(writer, err.Error(), http.StatusNotFound)
				return
			}
			log.Printf("Error creating order: %v", err)
			response.WriteServerError(writer)
			return
		}

		response.WriteJSON(writer, order, http.StatusCreated)
	})
}

type OrderWithItemIds struct {
	Id      string  `json:"order_id"`
	ItemIds []int32 `json:"items"`
	IsDone  bool    `json:"done"`
}

// createOrder creates a new order with the given item IDs and done status
func createOrder(ctx context.Context, db *sql.DB, queries *sqlc.Queries, itemIds []int32, isDone bool) (OrderWithItemIds, error) {
	tx, err := db.Begin()
	if err != nil {
		return OrderWithItemIds{}, err
	}

	defer tx.Rollback()

	qtx := queries.WithTx(tx)
	order, _ := qtx.CreateOrder(ctx, isDone)

	var pqErr *pq.Error
	var sliceItemIds []int32
	// iterating over item ids and creating new items
	for _, id := range itemIds {
		var itemId int32
		itemId, err = qtx.CreateOrderItems(ctx, sqlc.CreateOrderItemsParams{OrderID: order.ID, ItemID: id})
		if err != nil {
			if errors.As(err, &pqErr) {
				if pqErr.Code == "23503" {
					return OrderWithItemIds{}, &custom_errors.IdDoesNotExistError{Entity: "item", Id: strconv.Itoa(int(id))}
				}
			}

			return OrderWithItemIds{}, err
		}
		sliceItemIds = append(sliceItemIds, itemId)
	}

	if err = tx.Commit(); err != nil {
		return OrderWithItemIds{}, err
	}

	return OrderWithItemIds{Id: order.ID, ItemIds: sliceItemIds, IsDone: order.IsDone}, nil
}

// HandlerAddItemsToOrder handles the POST request to add items to an existing order
func HandlerAddItemsToOrder(ctx context.Context, db *sql.DB, queries *sqlc.Queries) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// Читаем тело запроса
		orderId := request.PathValue("order_id")

		body, err := io.ReadAll(request.Body)
		if err != nil {
			response.WritePlainText(writer, "Failed to read request body", http.StatusInternalServerError)
			return
		}
		defer request.Body.Close()

		// Определяем срез для хранения данных
		var itemIds []int32

		// Декодируем JSON
		err = json.Unmarshal(body, &itemIds)
		if err != nil {
			response.WritePlainText(writer, "Failed to decode JSON", http.StatusBadRequest)
			return
		}

		// adding items to the order
		err = addItemsToOrder(ctx, db, queries, orderId, itemIds)
		if err != nil {
			if errors.As(err, &custom_errors.ErrIdDoesNotExist) {
				response.WritePlainText(writer, err.Error(), http.StatusNotFound)
			} else if errors.As(err, &custom_errors.ErrOrderAlreadyDone) {
				response.WritePlainText(writer, err.Error(), http.StatusConflict)
			} else {
				response.WriteServerError(writer)
			}
			return
		}

		response.WriteOK(writer)
	})
}

// addItemsToOrder adds items to an existing order
func addItemsToOrder(ctx context.Context, db *sql.DB, queries *sqlc.Queries, orderId string, itemIds []int32) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	qtx := queries.WithTx(tx)
	order, err := qtx.GetOrder(ctx, orderId)
	if err != nil {
		return &custom_errors.IdDoesNotExistError{Entity: "order", Id: orderId}
	}

	if order.Done == true {
		return &custom_errors.OrderAlreadyDone{Id: orderId}
	}

	for _, id := range itemIds {
		_, err = qtx.CreateOrderItems(ctx, sqlc.CreateOrderItemsParams{OrderID: order.OrderID, ItemID: id})
		var pqErr *pq.Error
		if err != nil {
			if errors.As(err, &pqErr) {
				if pqErr.Code == "23503" {
					return &custom_errors.IdDoesNotExistError{Entity: "item", Id: strconv.Itoa(int(id))}
				}
			}

			return errors.New(fmt.Sprintf("could not add item with ID %d to order", id))
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

// HandlerMakeOrderDone handles the POST request to mark an order as done
func HandlerMakeOrderDone(ctx context.Context, db *sql.DB, queries *sqlc.Queries) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		orderId := request.PathValue("order_id")
		tx, err := db.Begin()
		defer tx.Rollback()

		if err != nil {
			response.WriteServerError(writer)
			return
		}

		qtx := queries.WithTx(tx)
		isDone, err := qtx.GetOrderStatus(ctx, orderId)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				response.WritePlainText(writer, "order not found", http.StatusNotFound)
				return
			}

			response.WriteServerError(writer)
			return
		}

		if isDone == true {
			response.WritePlainText(writer, "order is already done", http.StatusConflict)
			return
		}

		// setting order as done
		err = qtx.SetOrderDone(ctx, orderId)
		if err != nil {
			response.WriteServerError(writer)
			return
		}

		err = tx.Commit()
		if err != nil {
			response.WriteServerError(writer)
			return
		}

		response.WriteOK(writer)
	})
}

// HandlerListOrders handles the GET request to list orders
func HandlerListOrders(ctx context.Context, queries *sqlc.Queries) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		doneQuery := request.URL.Query().Get("done")

		// list all orders without filter
		if doneQuery == "" {
			orders, err := queries.ListOrders(ctx)
			if err != nil {
				response.WriteServerError(writer)
				return
			}

			response.WriteJSON(writer, orders, http.StatusOK)
			return
		} else if doneQuery == "1" {
			// list all done orders
			orders, err := queries.ListOrdersByDone(ctx, true)
			if err != nil {
				response.WriteServerError(writer)
				return
			}

			response.WriteJSON(writer, orders, http.StatusOK)
			return
		} else if doneQuery == "0" {
			// list all not done orders
			orders, err := queries.ListOrdersByDone(ctx, false)
			if err != nil {
				response.WriteServerError(writer)
				return
			}

			response.WriteJSON(writer, orders, http.StatusOK)
			return
		}

		response.WritePlainText(writer, "'done' query is an invalid value", http.StatusBadRequest)
	})
}
