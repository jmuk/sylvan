package openai

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jmuk/sylvan/pkg/chat/parts"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
)

func dataURL(blob *parts.Blob) string {
	encoded := base64.StdEncoding.EncodeToString(blob.Data)
	return fmt.Sprintf("data:%s,%s", blob.MimeType, encoded)
}

func textToInput(text string) responses.ResponseInputItemUnionParam {
	return responses.ResponseInputItemParamOfInputMessage(
		responses.ResponseInputMessageContentListParam{
			responses.ResponseInputContentParamOfInputText(text),
		},
		"user",
	)
}

func imageToInput(b *parts.Blob) responses.ResponseInputItemUnionParam {
	return responses.ResponseInputItemParamOfInputMessage(
		responses.ResponseInputMessageContentListParam{
			responses.ResponseInputContentUnionParam{
				OfInputImage: &responses.ResponseInputImageParam{
					Detail:   responses.ResponseInputImageDetailAuto,
					ImageURL: param.NewOpt(dataURL(b)),
					Type:     "input_image",
				},
			},
		},
		"user",
	)
}

func fileToInput(b *parts.Blob) responses.ResponseInputItemUnionParam {
	return responses.ResponseInputItemParamOfInputMessage(
		responses.ResponseInputMessageContentListParam{
			responses.ResponseInputContentUnionParam{
				OfInputFile: &responses.ResponseInputFileParam{
					FileData: param.NewOpt(string(b.Data)),
					Filename: param.NewOpt(b.Filename),
					Type:     "input_file",
				},
			},
		},
		"user",
	)
}

func funcRespToInput(fr *parts.FunctionResponse) (responses.ResponseInputItemUnionParam, error) {
	var output map[string]any
	if fr.Error != nil {
		output = map[string]any{
			"success":       false,
			"error_message": fr.Error.Error(),
		}
	} else {
		output = map[string]any{"success": true}
		if len(fr.Parts) > 0 {
			output["data"] = fr.Parts
		}
		if fr.Response != nil {
			output["response"] = fr.Response
		}
	}
	outputStr, err := json.Marshal(output)
	if err != nil {
		return responses.ResponseInputItemUnionParam{}, err
	}
	return responses.ResponseInputItemParamOfFunctionCallOutput(
		fr.ID,
		string(outputStr),
	), nil
}

func partToInput(p parts.Part, logger *slog.Logger) (responses.ResponseInputItemUnionParam, bool, error) {
	if p.Text != "" {
		return textToInput(p.Text), true, nil
	} else if p.Audio != nil {
		logger.Error("audio input is not supported", "part", p)
		return responses.ResponseInputItemUnionParam{}, false, nil
	} else if p.Image != nil {
		return imageToInput(p.Image), true, nil
	} else if p.File != nil {
		return fileToInput(p.File), true, nil
	} else if fr := p.FunctionResponse; fr != nil {
		resp, err := funcRespToInput(fr)
		return resp, err == nil, err
	}
	return responses.ResponseInputItemUnionParam{}, false, fmt.Errorf("unsupported part %+v", p)
}
