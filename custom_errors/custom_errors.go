package custom_errors

import (
	_ "errors"
	"fmt"
)

type IdDoesNotExistError struct {
	Entity string
	Id     string
}

func (e *IdDoesNotExistError) Error() string {
	return fmt.Sprintf("%s %s does not exist", e.Entity, e.Id)
}

var ErrIdDoesNotExist *IdDoesNotExistError

type OrderAlreadyDone struct {
	Id string
}

func (e *OrderAlreadyDone) Error() string {
	return fmt.Sprintf("order %s is already done", e.Id)
}

var ErrOrderAlreadyDone *OrderAlreadyDone
