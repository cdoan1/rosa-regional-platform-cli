package cloudformation

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/smithy-go"
)

// StackAlreadyExistsError is returned when attempting to create a stack that already exists
type StackAlreadyExistsError struct {
	StackName string
}

func (e *StackAlreadyExistsError) Error() string {
	return fmt.Sprintf("stack %s already exists", e.StackName)
}

// StackNotFoundError is returned when a stack cannot be found
type StackNotFoundError struct {
	StackName string
}

func (e *StackNotFoundError) Error() string {
	return fmt.Sprintf("stack %s not found", e.StackName)
}

// StackUpdateInProgressError is returned when attempting to update a stack that is currently being updated
type StackUpdateInProgressError struct {
	StackName string
}

func (e *StackUpdateInProgressError) Error() string {
	return fmt.Sprintf("stack %s is currently being updated", e.StackName)
}

// NoChangesError is returned when a stack update would result in no changes
type NoChangesError struct {
	StackName string
}

func (e *NoChangesError) Error() string {
	return fmt.Sprintf("stack %s has no changes to apply", e.StackName)
}

// wrapError wraps AWS CloudFormation errors into more user-friendly error types
func wrapError(err error) error {
	if err == nil {
		return nil
	}

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "AlreadyExistsException":
			// Extract stack name from error message if possible
			return &StackAlreadyExistsError{}
		case "ValidationError":
			if contains(apiErr.ErrorMessage(), "does not exist") {
				return &StackNotFoundError{}
			}
			if contains(apiErr.ErrorMessage(), "No updates are to be performed") {
				return &NoChangesError{}
			}
		case "ResourceNotFoundException":
			return &StackNotFoundError{}
		case "OperationInProgressException":
			return &StackUpdateInProgressError{}
		}
	}

	// Check for typed errors
	var stackNotFoundErr *types.StackNotFoundException
	if errors.As(err, &stackNotFoundErr) {
		return &StackNotFoundError{}
	}

	return err
}

// Helper function to check if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
