package oidc

import "fmt"

type OIDCError struct {
	Operation string
	Message   string
}

func (e *OIDCError) Error() string {
	return fmt.Sprintf("OIDC %s error: %s", e.Operation, e.Message)
}
