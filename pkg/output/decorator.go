package output

import (
	"fmt"
	"go/format"
)

// Decorator represents an interface to decorate the given content.
type Decorator interface {
	Decorate(content []byte) ([]byte, error)
}

// FormatDecorator applies a formatting decoration to the given content.
type FormatDecorator struct{}

// Decorate is the implementation of the Decorator interface.
func (FormatDecorator) Decorate(content []byte) ([]byte, error) {
	formatted, err := format.Source(content)
	if err != nil {
		return content, fmt.Errorf("could not format content: %w", err)
	}
	return formatted, nil
}
