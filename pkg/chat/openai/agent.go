package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"iter"
	"log/slog"

	"github.com/jmuk/sylvan/pkg/chat/parts"
	"github.com/jmuk/sylvan/pkg/session"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
)

type Agent struct {
	client      responses.ResponseService
	historyFile string

	model        shared.ResponsesModel
	systemPrompt string
	tools        []responses.ToolUnionParam

	previousResponseID param.Opt[string]
}

func dataURL(blob *parts.Blob) string {
	encoded := base64.StdEncoding.EncodeToString(blob.Data)
	return fmt.Sprintf("data:%s,%s", blob.MimeType, encoded)
}

func (a *Agent) SendMessageStream(ctx context.Context, ps []parts.Part) iter.Seq2[*parts.Part, error] {
	return func(yield func(*parts.Part, error) bool) {
		logger := slog.New(slog.DiscardHandler)
		if s, sok := session.FromContext(ctx); sok {
			if gotLogger, err := s.GetLogger("openai"); err != nil {
				if !yield(nil, err) {
					return
				}
			} else {
				logger = gotLogger
			}
		}

		var input responses.ResponseNewParamsInputUnion
		if len(ps) == 1 && ps[0].Text != "" {
			input = responses.ResponseNewParamsInputUnion{
				OfString: param.NewOpt(ps[0].Text),
			}
		} else {
			for _, p := range ps {
				var msg responses.ResponseInputItemUnionParam
				if p.Text != "" {
					msg = responses.ResponseInputItemParamOfInputMessage(
						responses.ResponseInputMessageContentListParam{
							responses.ResponseInputContentParamOfInputText(p.Text),
						},
						"user",
					)
				} else if p.Audio != nil {
					logger.Error("audio input is not supported", "part", p)
					continue
				} else if p.Image != nil {
					msg = responses.ResponseInputItemParamOfInputMessage(
						responses.ResponseInputMessageContentListParam{
							responses.ResponseInputContentUnionParam{
								OfInputImage: &responses.ResponseInputImageParam{
									Detail:   responses.ResponseInputImageDetailAuto,
									ImageURL: param.NewOpt(dataURL(p.Image)),
									Type:     "input_image",
								},
							},
						},
						"user",
					)
				} else if p.File != nil {
					msg = responses.ResponseInputItemParamOfInputMessage(
						responses.ResponseInputMessageContentListParam{
							responses.ResponseInputContentUnionParam{
								OfInputFile: &responses.ResponseInputFileParam{
									FileData: param.NewOpt(string(p.File.Data)),
									Filename: param.NewOpt(p.File.Filename),
									Type:     "input_file",
								},
							},
						},
						"user",
					)
				} else if fr := p.FunctionResponse; fr != nil {
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
						if !yield(nil, err) {
							return
						}
					}
					msg = responses.ResponseInputItemParamOfFunctionCallOutput(
						p.FunctionResponse.ID,
						string(outputStr),
					)
				}
				input.OfInputItemList = append(input.OfInputItemList, msg)
			}
		}
		// TODO: support streaming for better UX.
		resp, err := a.client.New(ctx, responses.ResponseNewParams{
			Instructions:       param.NewOpt(a.systemPrompt),
			PreviousResponseID: a.previousResponseID,
			Input:              input,
			Model:              a.model,
			Tools:              a.tools,
		})
		if err != nil {
			yield(nil, err)
			return
		}
		a.previousResponseID = param.NewOpt(resp.ID)
		for _, output := range resp.Output {
			switch output.Type {
			case "message":
				msg := output.AsMessage()
				for _, content := range msg.Content {
					if !yield(&parts.Part{
						Text: content.Text,
					}, nil) {
						return
					}
				}
			case "reasoning":
				reason := output.AsReasoning()
				for _, content := range reason.Content {
					if !yield(&parts.Part{
						Text:    content.Text,
						Thought: true,
					}, nil) {
						return
					}
				}
			case "function_call":
				fc := output.AsFunctionCall()
				parsedArgs := map[string]any{}
				if err := json.Unmarshal([]byte(fc.Arguments), &parsedArgs); err != nil {
					if !yield(nil, err) {
						return
					}
				} else if !yield(&parts.Part{
					FunctionCall: &parts.FunctionCall{
						ID:   fc.CallID,
						Name: fc.Name,
						Args: parsedArgs,
					},
				}, nil) {
					return
				}
			default:
				logger.Warn("unsupported output", "type", output.Type, "output", output)
			}
		}
	}
}
