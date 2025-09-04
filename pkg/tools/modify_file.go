package tools

import (
	"context"
	"fmt"
	"sort"

	"github.com/sergi/go-diff/diffmatchpatch"
)

type modification struct {
	Offset   int64  `json:"offset" jsonschema:"required,description=the offset starting point in bytes to be changed"`
	Previous string `json:"previous" jsonschema:"required,description=the content before the modification"`
	Replace  string `json:"replace" jsonschema:"required,description=the new content to replace the previous content"`
}

type modifyFileRequest struct {
	Filename      string         `json:"filename" jsonschema:"required"`
	Modifications []modification `json:"modifications" jsonschema:"the list of changes to make"`
}

type modifyFileResponse struct {
	Ok         bool   `json:"ok" jsonschema:"required,description=whether all the modification has been made or not"`
	Error      string `json:"error" jsonschema:"description=the error message for the first modification to fail or empty if it succeeds"`
	NewContent string `json:"new_content" jsonschema:"description=the content modified by the user and actually saved"`
}

func (ft *FileTools) modifyFile(ctx context.Context, req modifyFileRequest) modifyFileResponse {
	logger := getLogger(ctx)
	logger.Info("Modify file")
	data, err := ft.root.ReadFile(req.Filename)
	if err != nil {
		logger.Error("Error reading file", "error", err)
		return modifyFileResponse{
			Ok:    false,
			Error: err.Error(),
		}
	}
	strData := string(data)

	type modWithIndex struct {
		modification
		index int
	}
	sortedMods := make([]modWithIndex, 0, len(req.Modifications))
	for i, m := range req.Modifications {
		sortedMods = append(sortedMods, modWithIndex{
			modification: m,
			index:        i,
		})
	}
	sort.Slice(sortedMods, func(i, j int) bool {
		m1 := sortedMods[i]
		m2 := sortedMods[j]
		return m1.Offset > m2.Offset
	})

	// modify from the last for the simplicity of the edit.
	for _, m := range sortedMods {
		mlog := logger.With(
			"index", m.index, "offset", m.Offset,
			"previous", m.Previous, "replace", m.Replace)
		mlog.Debug("Modification")
		if len(data) < int(m.Offset) || m.Offset < 0 {
			mlog.Error("Invalid modification")
			return modifyFileResponse{
				Ok: false,
				Error: fmt.Sprintf(
					"invalid offset %d at %d-th modification", m.Offset, m.index),
			}
		}
		start := int(m.Offset)
		end := start + len(m.Previous)
		substr := strData[start:end]
		if substr != m.Previous {
			mlog.Error("Previous does not match", "substr", substr)
			return modifyFileResponse{
				Ok: false,
				Error: fmt.Sprintf(
					"previous %s does not match with the content %s at %d-th modification",
					m.Previous, substr, m.index),
			}
		}
		strData = strData[:start] + m.Replace + strData[end:]
	}

	withNewContent := false
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(string(data), strData, false)
	fmt.Printf("Modifying %s as:", req.Filename)
	fmt.Println(dmp.DiffPrettyText(diffs))
	answer, err := confirm()
	if err != nil {
		logger.Error("Failed to confirm", "error", err)
		return modifyFileResponse{
			Ok:    false,
			Error: err.Error(),
		}
	}
	if answer == confirmationEdit {
		strData, err = userEdit(logger, req.Filename, strData)
		if err != nil {
			logger.Error("Failed to confirm", "error", err)
			return modifyFileResponse{
				Ok:    false,
				Error: err.Error(),
			}
		}
		withNewContent = true
	} else if answer != confirmationYes {
		logger.Error("User declined")
		return modifyFileResponse{
			Ok:    false,
			Error: "User declined to accept the change",
		}
	}

	if err := ft.root.WriteFile(req.Filename, []byte(strData), 0644); err != nil {
		logger.Error("Failed to write the file", "error", err)
		return modifyFileResponse{
			Ok:    false,
			Error: err.Error(),
		}
	}
	resp := modifyFileResponse{
		Ok: true,
	}
	if withNewContent {
		resp.NewContent = strData
	}
	return resp
}
