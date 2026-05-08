package output

import (
	"os"
	"path/filepath"
)

const (
	// FileWriterExtension is the extension to write files of.
	FileWriterExtension = ".go"
)

// Writer represents an interface to write the produced struct content.
type Writer interface {
	Write(tableName string, content []byte) error
}

// FileWriter is a writer that writes to a file given by the path and the table name.
type FileWriter struct {
	path       string
	decorators []Decorator
}

// NewFileWriter constructs a new FileWriter.
func NewFileWriter(path string) *FileWriter {
	path = filepath.Clean(path)
	if !os.IsPathSeparator(path[len(path)-1]) {
		path += string(os.PathSeparator)
	}
	return &FileWriter{
		path: path,
		decorators: []Decorator{
			FormatDecorator{},
		},
	}
}

// Write is the implementation of the Writer interface. The FileWriter writes
// decorated content to the file specified by the given path and table name.
func (w *FileWriter) Write(tableName string, content []byte) error {
	fileName := w.path + tableName + FileWriterExtension

	decorated, err := w.decorate(content)
	if err != nil {
		return err
	}

	return os.WriteFile(fileName, decorated, 0666)
}

// decorate applies decorations like formatting.
func (w *FileWriter) decorate(content []byte) (decorated []byte, err error) {
	for i := range w.decorators {
		content, err = w.decorators[i].Decorate(content)
		if err != nil {
			return content, err
		}
	}

	return content, nil
}
