package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatDecorator_Decorate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc     string
		input    []byte
		expected []byte
		isError  assert.ErrorAssertionFunc
	}{
		{
			desc:     "well formed golang code should get decorated",
			input:    []byte("package dto\ntype Bar struct {\nID int `db:\"id\"`\n}"),
			expected: []byte("package dto\n\ntype Bar struct {\n\tID int `db:\"id\"`\n}\n"),
			isError:  assert.NoError,
		},
		{
			desc:     "arbitrary text throws error",
			input:    []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit"),
			expected: []byte{},
			isError:  assert.Error,
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			decorator := FormatDecorator{}
			actual, err := decorator.Decorate(test.input)
			if err != nil {
				test.isError(t, err)
				return
			}
			assert.Equal(t, test.expected, actual)
		})
	}
}
