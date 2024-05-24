package model

import "fmt"

type CustomError struct {
	Code    int
	Message string
}

func (e *CustomError) Error() string {
	return fmt.Sprintf("Error Code: %d, Message: %s", e.Code, e.Message)
}
