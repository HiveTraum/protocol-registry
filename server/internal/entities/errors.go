package entities

import (
	"fmt"
	"strings"
)

// ErrorCode represents domain error codes
type ErrorCode string

const (
	ErrorCodeSyntaxError             ErrorCode = "SYNTAX_ERROR"
	ErrorCodeBreakingChanges         ErrorCode = "BREAKING_CHANGES"
	ErrorCodeServiceNotFound         ErrorCode = "SERVICE_NOT_FOUND"
	ErrorCodeProtocolNotFound        ErrorCode = "PROTOCOL_NOT_FOUND"
	ErrorCodeConsumerNotFound        ErrorCode = "CONSUMER_NOT_FOUND"
	ErrorCodeConsumerBreakingChanges ErrorCode = "CONSUMER_BREAKING_CHANGES"
)

// DomainError is the base domain error type
type DomainError struct {
	Code    ErrorCode
	Message string
	Details any
}

func (e *DomainError) Error() string {
	return e.Message
}

func (e *DomainError) Is(target error) bool {
	t, ok := target.(*DomainError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// Sentinel errors for errors.Is() checks
var (
	ErrServiceNotFound  = &DomainError{Code: ErrorCodeServiceNotFound, Message: "service not found"}
	ErrProtocolNotFound = &DomainError{Code: ErrorCodeProtocolNotFound, Message: "protocol not found"}
	ErrConsumerNotFound = &DomainError{Code: ErrorCodeConsumerNotFound, Message: "consumer not found"}
)

// ConsumerBreakingChange represents breaking changes caused by a specific consumer
type ConsumerBreakingChange struct {
	ConsumerName string
	Changes      []BreakingChange
}

// BreakingChange represents a single breaking change violation
type BreakingChange struct {
	Type    BreakingChangeType
	Subject string
	Message string
}

type BreakingChangeType string

const (
	BreakingChangeMessageRemoved          BreakingChangeType = "MESSAGE_REMOVED"
	BreakingChangeFieldRemoved            BreakingChangeType = "FIELD_REMOVED"
	BreakingChangeFieldTypeChanged        BreakingChangeType = "FIELD_TYPE_CHANGED"
	BreakingChangeFieldCardinalityChanged BreakingChangeType = "FIELD_CARDINALITY_CHANGED"
	BreakingChangeFieldRenamed            BreakingChangeType = "FIELD_RENAMED"
	BreakingChangeEnumRemoved             BreakingChangeType = "ENUM_REMOVED"
	BreakingChangeEnumValueRemoved        BreakingChangeType = "ENUM_VALUE_REMOVED"
	BreakingChangeServiceRemoved          BreakingChangeType = "SERVICE_REMOVED"
	BreakingChangeMethodRemoved           BreakingChangeType = "METHOD_REMOVED"
	BreakingChangeMethodInputChanged      BreakingChangeType = "METHOD_INPUT_CHANGED"
	BreakingChangeMethodOutputChanged     BreakingChangeType = "METHOD_OUTPUT_CHANGED"
	BreakingChangeMethodStreamingChanged  BreakingChangeType = "METHOD_STREAMING_CHANGED"

	// OpenAPI breaking change types
	BreakingChangeEndpointRemoved        BreakingChangeType = "ENDPOINT_REMOVED"
	BreakingChangeHTTPMethodRemoved      BreakingChangeType = "HTTP_METHOD_REMOVED"
	BreakingChangePathParameterRemoved   BreakingChangeType = "PATH_PARAMETER_REMOVED"
	BreakingChangeRequestFieldRemoved    BreakingChangeType = "REQUEST_FIELD_REMOVED"
	BreakingChangeRequestFieldTypeChanged BreakingChangeType = "REQUEST_FIELD_TYPE_CHANGED"
	BreakingChangeResponseFieldRemoved   BreakingChangeType = "RESPONSE_FIELD_REMOVED"
	BreakingChangeResponseFieldTypeChanged BreakingChangeType = "RESPONSE_FIELD_TYPE_CHANGED"
)

// NewBreakingChangesError creates a domain error for breaking changes
func NewBreakingChangesError(changes []BreakingChange) *DomainError {
	var msgs []string
	for _, c := range changes {
		msgs = append(msgs, fmt.Sprintf("[%s] %s", c.Type, c.Message))
	}

	return &DomainError{
		Code:    ErrorCodeBreakingChanges,
		Message: fmt.Sprintf("breaking changes detected (%d):\n%s", len(changes), strings.Join(msgs, "\n")),
		Details: changes,
	}
}

// NewSyntaxError creates a domain error for syntax errors
func NewSyntaxError(errors []string) *DomainError {
	return &DomainError{
		Code:    ErrorCodeSyntaxError,
		Message: fmt.Sprintf("syntax errors (%d):\n%s", len(errors), strings.Join(errors, "\n")),
		Details: errors,
	}
}

// NewServiceNotFoundError creates a domain error for service not found
func NewServiceNotFoundError(serviceName string) *DomainError {
	return &DomainError{
		Code:    ErrorCodeServiceNotFound,
		Message: fmt.Sprintf("service %q not found", serviceName),
	}
}

// NewProtocolNotFoundError creates a domain error for protocol not found
func NewProtocolNotFoundError(serviceName string) *DomainError {
	return &DomainError{
		Code:    ErrorCodeProtocolNotFound,
		Message: fmt.Sprintf("protocol not found for service %q", serviceName),
	}
}

// BreakingChanges returns breaking changes if this is a breaking changes error
func (e *DomainError) BreakingChanges() []BreakingChange {
	if e.Code != ErrorCodeBreakingChanges {
		return nil
	}
	if changes, ok := e.Details.([]BreakingChange); ok {
		return changes
	}
	return nil
}

// SyntaxErrors returns syntax errors if this is a syntax error
func (e *DomainError) SyntaxErrors() []string {
	if e.Code != ErrorCodeSyntaxError {
		return nil
	}
	if errs, ok := e.Details.([]string); ok {
		return errs
	}
	return nil
}

// NewConsumerNotFoundError creates a domain error for consumer not found
func NewConsumerNotFoundError(consumerName, serverName string) *DomainError {
	return &DomainError{
		Code:    ErrorCodeConsumerNotFound,
		Message: fmt.Sprintf("consumer %q not found for server %q", consumerName, serverName),
	}
}

// NewConsumerBreakingChangesError creates a domain error for breaking changes affecting consumers
func NewConsumerBreakingChangesError(violations []ConsumerBreakingChange) *DomainError {
	var msgs []string
	for _, v := range violations {
		for _, c := range v.Changes {
			msgs = append(msgs, fmt.Sprintf("consumer %q: [%s] %s", v.ConsumerName, c.Type, c.Message))
		}
	}

	return &DomainError{
		Code:    ErrorCodeConsumerBreakingChanges,
		Message: fmt.Sprintf("breaking changes affect consumers (%d):\n%s", len(violations), strings.Join(msgs, "\n")),
		Details: violations,
	}
}

// ConsumerBreakingChanges returns consumer breaking changes if this is a consumer breaking changes error
func (e *DomainError) ConsumerBreakingChanges() []ConsumerBreakingChange {
	if e.Code != ErrorCodeConsumerBreakingChanges {
		return nil
	}
	if violations, ok := e.Details.([]ConsumerBreakingChange); ok {
		return violations
	}
	return nil
}
