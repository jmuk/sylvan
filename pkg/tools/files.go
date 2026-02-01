package tools

import (
	"context"
	"os"
)

// FileTools provides the tools/functions related to files
// (read/write/update/search).  It is guarded under a
// certain directory (typically the current directory of
// the command started).
type FileTools struct {
	root     *os.Root
	rootPath string
}

// NewFiles creates a new FileTools in the directory.
func NewFiles(rootPath string) *FileTools {
	return &FileTools{
		rootPath: rootPath,
	}
}

func (ft *FileTools) getRoot() (*os.Root, error) {
	if ft.root != nil {
		return ft.root, nil
	}
	root, err := os.OpenRoot(ft.rootPath)
	if err != nil {
		return nil, err
	}
	ft.root = root
	return ft.root, nil
}

// Close closes the handle to the root.
func (ft *FileTools) Close() error {
	if ft.root == nil {
		return nil
	}
	root := ft.root
	ft.root = nil
	return root.Close()
}

// ToolDefs implements Manager interface.
func (ft *FileTools) ToolDefs(ctx context.Context) ([]ToolDefinition, error) {
	return []ToolDefinition{
		&toolDefinition[readFileRequest, *readFileResponse]{
			name:        "read_file",
			description: "Read a file",
			proc:        ft.readFile,
		},
		&toolDefinition[searchFilesRequest, *searchFilesResponse]{
			name:        "search_files",
			description: "return the list of file paths matching with the path patterns or contents",
			proc:        ft.searchFile,
		},
		&toolDefinition[writeFileRequest, string]{
			name:            "write_file",
			description:     "Write the content to a file; overwriting an existing one or create a new file",
			proc:            ft.writeFile,
			respName:        "new_content",
			respDescription: "the content to be stored in the file in case it's different from the request",
		},
		&toolDefinition[modifyFileRequest, string]{
			name:            "modify_file",
			description:     "modify the contents of a file",
			proc:            ft.modifyFile,
			respName:        "new_content",
			respDescription: "the content to be stored in the file in case it's different from the request",
		},
		&toolDefinition[deleteFileRequest, *deleteFileResponse]{
			name:        "delete_file",
			description: "delete a file",
			proc:        ft.deleteFile,
		},
		&toolDefinition[createDirRequest, *createDirResponse]{
			name:        "create_directory",
			description: "create a new directory",
			proc:        ft.createDir,
		},
	}, nil
}
