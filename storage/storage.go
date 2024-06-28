package storage

import (
	"context"
	"pizza-factory-go/dto"
)

type OrderStorage interface {
	GetOrderWithItemIds(ctx context.Context, orderId string) (dto.OrderWithItemIds, error)
	CreateOrder(ctx context.Context, itemIds []int32, isDone bool) (dto.OrderWithItemIds, error)
	AddItemsToOrder(ctx context.Context, orderId string, itemIds []int32) error
	MakeOrderDone(ctx context.Context, orderId string) error
	ListOrders(ctx context.Context) (orders []dto.Order, err error)
	ListOrdersByDone(ctx context.Context, isDone bool) (orders []dto.Order, err error)
}
