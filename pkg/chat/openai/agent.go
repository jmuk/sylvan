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
			logger.Debug("input", "input", input)
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
		logger.Debug("sending", "input", input)
		st := a.client.NewStreaming(ctx, responses.ResponseNewParams{
			Instructions:       param.NewOpt(a.systemPrompt),
			PreviousResponseID: a.previousResponseID,
			Input:              input,
			Model:              a.model,
			Tools:              a.tools,
		})

		var fc *parts.FunctionCall
		var funcCallParam string
		for st.Next() {
			ev := st.Current()
			logger.Debug("Received event", "event", ev)
			switch variant := ev.AsAny().(type) {
			case responses.ResponseCreatedEvent:
				a.previousResponseID = param.NewOpt(variant.Response.ID)
			case responses.ResponseErrorEvent:
				err := fmt.Errorf("failed: %s %s %s", variant.Code, variant.Message, variant.Param)
				if !yield(nil, err) {
					return
				}
			case responses.ResponseTextDeltaEvent:
				if !yield(&parts.Part{Text: variant.Delta}, nil) {
					return
				}
			case responses.ResponseTextDoneEvent:
				if !yield(&parts.Part{Text: variant.Text}, nil) {
					return
				}
			case responses.ResponseReasoningTextDeltaEvent:
				if !yield(&parts.Part{Text: variant.Delta, Thought: true}, nil) {
					return
				}
			case responses.ResponseReasoningTextDoneEvent:
				if !yield(&parts.Part{Text: variant.Text, Thought: true}, nil) {
					return
				}
			case responses.ResponseOutputItemAddedEvent:
				if variant.Item.Type == "function_call" {
					call := variant.Item.AsFunctionCall()
					fc = &parts.FunctionCall{
						Name: call.Name,
						ID:   call.CallID,
					}
					funcCallParam = call.Arguments
				}
			case responses.ResponseFunctionCallArgumentsDeltaEvent:
				funcCallParam += variant.Delta
			case responses.ResponseFunctionCallArgumentsDoneEvent:
				if fc == nil {
					continue
				}
				fc.Args = map[string]any{}
				if err := json.Unmarshal([]byte(funcCallParam), &fc.Args); err != nil {
					funcCallParam = ""
					if !yield(nil, err) {
						return
					}
				}
				funcCallParam = ""
				if !yield(&parts.Part{FunctionCall: fc}, nil) {
					return
				}
				fc = nil
			default:
				logger.Error("unknown response")
			}
		}
		if st.Err() != nil {
			yield(nil, st.Err())
		}
	}
}
