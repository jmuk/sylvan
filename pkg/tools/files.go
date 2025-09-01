package tools

import (
	"log/slog"
	"os"
)

// FileTools provides the tools/functions related to files
// (read/write/update/search).  It is guarded under a
// certain directory (typically the current directory of
// the command started).
type FileTools struct {
	root   *os.Root
	logger *slog.Logger
}

// NewFiles creates a new FileTools in the directory.
func NewFiles(rootPath string, handler slog.Handler) (*FileTools, error) {
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, err
	}
	return &FileTools{
		root:   root,
		logger: slog.New(handler),
	}, nil
}

// Close closes the handle to the root.
func (ft *FileTools) Close() error {
	return ft.root.Close()
}

func (ft *FileTools) ToolDefs() []ToolDefinition {
	return []ToolDefinition{
		&toolDefinition[readFileRequest, readFileResponse]{
			name:        "read_file",
			description: "Read a file",
			proc:        ft.readFile,
		},
		&toolDefinition[searchFilesRequest, searchFilesResponse]{
			name:        "search_files",
			description: "return the list of file paths matching with the path patterns or contents",
			proc:        ft.searchFile,
		},
		&toolDefinition[writeFileRequest, writeFileResponse]{
			name:        "write_file",
			description: "Write the content to a file; overwriting an existing one or create a new file",
			proc:        ft.writeFile,
		},
		&toolDefinition[modifyFileRequest, modifyFileResponse]{
			name:        "modify_file",
			description: "modify the contents of a file",
			proc:        ft.modifyFile,
		},
	}
}
