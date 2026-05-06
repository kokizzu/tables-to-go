package output

import (
	"fmt"
	"go/format"
)

// Decorator represents an interface to decorate the given content.
type Decorator interface {
	Decorate(content string) (string, error)
}

// FormatDecorator applies a formatting decoration to the given content.
type FormatDecorator struct{}

// Decorate is the implementation of the Decorator interface.
func (FormatDecorator) Decorate(content string) (string, error) {
	formatted, err := format.Source([]byte(content))
	if err != nil {
		return content, fmt.Errorf("could not format content: %w", err)
	}
	return string(formatted), nil
}
