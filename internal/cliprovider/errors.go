package cliprovider

import (
	"errors"
	"fmt"
)

type ErrorCode string

const (
	CodeInvalidInput ErrorCode = "invalid_input"
	CodeUnavailable  ErrorCode = "unavailable"
	CodeTimeout      ErrorCode = "timeout"
	CodeCancelled    ErrorCode = "cancelled"
	CodeProcess      ErrorCode = "process_failed"
	CodeInvalidReply ErrorCode = "invalid_reply"
)

// Error deliberately excludes prompts, generated output, local paths, stderr,
// credentials, and provider account details from its public message.
type Error struct {
	Provider Provider  `json:"provider"`
	Stage    Stage     `json:"stage,omitempty"`
	Code     ErrorCode `json:"code"`
	Message  string    `json:"message"`
}

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	if err.Provider == "" && err.Stage == "" {
		return err.Message
	}
	if err.Provider == "" {
		return fmt.Sprintf("%s: %s", err.Stage, err.Message)
	}
	if err.Stage != "" {
		return fmt.Sprintf("%s/%s: %s", err.Provider, err.Stage, err.Message)
	}
	return fmt.Sprintf("%s: %s", err.Provider, err.Message)
}

func IsCode(err error, code ErrorCode) bool {
	var providerErr *Error
	return errors.As(err, &providerErr) && providerErr.Code == code
}

func providerError(provider Provider, code ErrorCode, message string, _ error) error {
	return &Error{Provider: provider, Code: code, Message: message}
}

func stageError(provider Provider, stage Stage, code ErrorCode, message string, _ error) error {
	return &Error{Provider: provider, Stage: stage, Code: code, Message: message}
}
