package s3

import "fmt"

type S3Error struct {
	Operation string
	Message   string
}

func (e *S3Error) Error() string {
	return fmt.Sprintf("S3 %s error: %s", e.Operation, e.Message)
}
