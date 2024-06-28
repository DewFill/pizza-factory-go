package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"pizza-factory-go/custom_errors"
	"pizza-factory-go/dto"
	"pizza-factory-go/response"
	"pizza-factory-go/storage"
)

// HandlerGetOrder handles the GET request to retrieve an order by its ID
func HandlerGetOrder(ctx context.Context, storageOrder storage.OrderStorage) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		orderId := request.PathValue("order_id")
		order, err := storageOrder.GetOrderWithItemIds(ctx, orderId)

		if err != nil {
			response.WritePlainText(writer, "order not found", http.StatusNotFound)
			return
		}

		log.Println("Returning order", order)

		response.WriteJSON(writer, order, http.StatusOK)
	})
}

// HandlerCreateOrder handles the POST request to create a new order
func HandlerCreateOrder(ctx context.Context, orderStorage storage.OrderStorage) http.Handler {
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
		order, err := orderStorage.CreateOrder(ctx, data.ItemIds, false)
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

// HandlerAddItemsToOrder handles the POST request to add items to an existing order
func HandlerAddItemsToOrder(ctx context.Context, orderStorage storage.OrderStorage) http.Handler {
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
		err = orderStorage.AddItemsToOrder(ctx, orderId, itemIds)
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

// HandlerMakeOrderDone handles the POST request to mark an order as done
func HandlerMakeOrderDone(ctx context.Context, orderStorage storage.OrderStorage) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		orderId := request.PathValue("order_id")
		err := orderStorage.MakeOrderDone(ctx, orderId)
		if err != nil {
			if errors.As(err, &custom_errors.ErrOrderAlreadyDone) {
				response.WritePlainText(writer, err.Error(), http.StatusConflict)
				return
			} else if errors.As(err, &custom_errors.ErrIdDoesNotExist) {
				response.WritePlainText(writer, err.Error(), http.StatusNotFound)
				return
			} else {
				response.WriteServerError(writer)
				return
			}
		}

		response.WriteOK(writer)
	})
}

// HandlerListOrders handles the GET request to list orders
func HandlerListOrders(ctx context.Context, orderStorage storage.OrderStorage) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		doneQuery := request.URL.Query().Get("done")

		var orders []dto.Order
		var err error
		// list all orders without filter
		if doneQuery == "" {
			orders, err = orderStorage.ListOrders(ctx)
		} else if doneQuery == "1" {
			// list all done orders
			orders, err = orderStorage.ListOrdersByDone(ctx, true)
		} else if doneQuery == "0" {
			// list all not done orders
			orders, err = orderStorage.ListOrdersByDone(ctx, false)
		} else {
			response.WritePlainText(writer, "The 'done' query has an invalid value.", http.StatusBadRequest)
			return
		}

		if err != nil {
			response.WriteServerError(writer)
			return
		}

		response.WriteJSON(writer, orders, http.StatusOK)
		return
	})
}
