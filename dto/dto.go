package dto

type OrderWithItemIds struct {
	ID      string  `json:"order_id"`
	ItemIds []int32 `json:"items"`
	IsDone  bool    `json:"done"`
}

type Order struct {
	ID     string `json:"order_id"`
	IsDone bool   `json:"done"`
}
