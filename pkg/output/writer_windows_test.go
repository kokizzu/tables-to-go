//go:build windows

package output

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFileWriter(t *testing.T) {
	t.Parallel()

	t.Logf("TestNewFileWriter under %s/%s", runtime.GOOS, runtime.GOARCH)

	tests := []struct {
		desc     string
		path     string
		expected *FileWriter
	}{
		{
			desc: "empty path should resolve to current directory with trailing separator",
			path: "",
			expected: &FileWriter{
				path: `.\`,
				decorators: []Decorator{
					FormatDecorator{},
				},
			},
		},
		{
			desc: "dot path should keep current directory with trailing separator",
			path: ".",
			expected: &FileWriter{
				path: `.\`,
				decorators: []Decorator{
					FormatDecorator{},
				},
			},
		},
		{
			desc: "relative path should be cleaned and end with one separator",
			path: `foo\bar\..\baz`,
			expected: &FileWriter{
				path: `foo\baz\`,
				decorators: []Decorator{
					FormatDecorator{},
				},
			},
		},
		{
			desc: "windows absolute path should be cleaned and end with separator",
			path: `C:\foo\..\bar`,
			expected: &FileWriter{
				path: `C:\bar\`,
				decorators: []Decorator{
					FormatDecorator{},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			actual := NewFileWriter(test.path)
			assert.Equal(t, test.expected, actual)
		})
	}
}
