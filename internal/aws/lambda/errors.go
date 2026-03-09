package lambda

import "fmt"

type LambdaError struct {
	Operation string
	Message   string
}

func (e *LambdaError) Error() string {
	return fmt.Sprintf("Lambda %s error: %s", e.Operation, e.Message)
}
