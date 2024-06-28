package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/lib/pq"
	"log"
	"pizza-factory-go/custom_errors"
	"pizza-factory-go/dto"
	"pizza-factory-go/sqlc"
	"strconv"
)

type OrderStorage struct {
	db      *sql.DB
	queries *sqlc.Queries
}

func NewOrderStorage(db *sql.DB, queries *sqlc.Queries) *OrderStorage {
	return &OrderStorage{db: db, queries: queries}
}

func (o OrderStorage) GetOrderWithItemIds(ctx context.Context, orderId string) (dto.OrderWithItemIds, error) {
	order, err := o.queries.GetOrderWithItemIds(ctx, orderId)
	if err != nil {
		return dto.OrderWithItemIds{}, err
	}

	orderModel := dto.OrderWithItemIds{
		ID:      order.ID,
		IsDone:  order.IsDone,
		ItemIds: order.ItemIds,
	}

	log.Println(order, err)

	return orderModel, nil
}

func (o OrderStorage) CreateOrder(ctx context.Context, itemIds []int32, isDone bool) (dto.OrderWithItemIds, error) {
	tx, err := o.db.Begin()
	if err != nil {
		return dto.OrderWithItemIds{}, err
	}

	defer tx.Rollback()

	qtx := o.queries.WithTx(tx)
	order, err := qtx.CreateOrder(ctx, isDone)

	if err != nil {
		return dto.OrderWithItemIds{}, err
	}

	var pqErr *pq.Error
	var sliceItemIds []int32
	// iterating over item ids and creating new items
	for _, id := range itemIds {
		var itemId int32
		itemId, err = qtx.CreateOrderItems(ctx, sqlc.CreateOrderItemsParams{OrderID: order.ID, ItemID: id})
		if err != nil {
			if errors.As(err, &pqErr) {
				if pqErr.Code == "23503" {
					return dto.OrderWithItemIds{}, &custom_errors.IdDoesNotExistError{Entity: "item", Id: strconv.Itoa(int(id))}
				}
			}

			return dto.OrderWithItemIds{}, err
		}
		sliceItemIds = append(sliceItemIds, itemId)
	}
	tx.Commit()

	//log.Println(sliceItemIds)

	return dto.OrderWithItemIds{ID: order.ID, ItemIds: sliceItemIds, IsDone: order.IsDone}, nil
}

func (o OrderStorage) AddItemsToOrder(ctx context.Context, orderId string, itemIds []int32) error {
	tx, err := o.db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	qtx := o.queries.WithTx(tx)
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

func (o OrderStorage) MakeOrderDone(ctx context.Context, orderId string) error {
	tx, err := o.db.Begin()
	defer tx.Rollback()

	if err != nil {
		return errors.New(fmt.Sprintf("could not make order done: %s", err))
	}

	qtx := o.queries.WithTx(tx)
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

func (o OrderStorage) ListOrders(ctx context.Context) ([]dto.Order, error) {
	orders, err := o.queries.ListOrders(ctx)
	if err != nil {
		return nil, err
	}

	ordersDTO := make([]dto.Order, 0)

	for _, i2 := range orders {
		ordersDTO = append(ordersDTO, dto.Order{ID: i2.OrderID, IsDone: i2.Done})
	}

	return ordersDTO, nil
}

func (o OrderStorage) ListOrdersByDone(ctx context.Context, isDone bool) ([]dto.Order, error) {
	orders, err := o.queries.ListOrdersByDone(ctx, isDone)
	if err != nil {
		return nil, err
	}

	ordersDTO := make([]dto.Order, 0)

	for _, i2 := range orders {
		ordersDTO = append(ordersDTO, dto.Order{ID: i2.OrderID, IsDone: i2.Done})
	}

	return ordersDTO, nil
}
