package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/lib/pq"
	"pizza-factory-go/custom_errors"
	"pizza-factory-go/sqlc"
	"strconv"
)

type OrderService struct {
	db      *sql.DB
	queries *sqlc.Queries
}

func NewOrderService(db *sql.DB, queries *sqlc.Queries) *OrderService {
	return &OrderService{db: db, queries: queries}
}

type OrderWithItemIds struct {
	Id      string  `json:"order_id"`
	ItemIds []int32 `json:"items"`
	IsDone  bool    `json:"done"`
}

// GetOrder returns order by ID
func (service *OrderService) GetOrder(ctx context.Context, orderId string) (order sqlc.GetOrderWithItemIdsRow, err error) {
	order, err = service.queries.GetOrderWithItemIds(ctx, orderId)
	return
}

// CreateOrder creates a new order with the given item IDs and done status
func (service *OrderService) CreateOrder(ctx context.Context, itemIds []int32, isDone bool) (OrderWithItemIds, error) {
	tx, err := service.db.Begin()
	if err != nil {
		return OrderWithItemIds{}, err
	}

	defer tx.Rollback()

	qtx := service.queries.WithTx(tx)
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

// AddItemsToOrder adds items to an existing order
func (service *OrderService) AddItemsToOrder(ctx context.Context, orderId string, itemIds []int32) error {
	tx, err := service.db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	qtx := service.queries.WithTx(tx)
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

// MakeOrderDone Sets order as complete
func (service *OrderService) MakeOrderDone(ctx context.Context, orderId string) error {
	tx, err := service.db.Begin()
	defer tx.Rollback()

	if err != nil {
		return errors.New(fmt.Sprintf("could not make order done: %s", err))
	}

	qtx := service.queries.WithTx(tx)
	isDone, err := qtx.GetOrderStatus(ctx, orderId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &custom_errors.IdDoesNotExistError{Entity: "order", Id: orderId}
		}

		return errors.New(fmt.Sprintf("could not make order done: %s", err))
	}

	if isDone == true {
		return &custom_errors.OrderAlreadyDone{Id: orderId}
	}

	// setting order as done
	err = qtx.SetOrderDone(ctx, orderId)
	if err != nil {
		return errors.New(fmt.Sprintf("could not make order done: %s", err))
	}

	err = tx.Commit()
	if err != nil {
		return errors.New(fmt.Sprintf("could not make order done: %s", err))
	}

	return nil
}

// ListOrders Returns all orders
func (service *OrderService) ListOrders(ctx context.Context) (orders []sqlc.ListOrdersRow, err error) {
	orders, err = service.queries.ListOrders(ctx)
	return
}

// ListOrdersByDone Returns a list of orders filtered by completion
func (service *OrderService) ListOrdersByDone(ctx context.Context, isDone bool) (orders []sqlc.ListOrdersByDoneRow, err error) {
	orders, err = service.queries.ListOrdersByDone(ctx, isDone)
	return
}
