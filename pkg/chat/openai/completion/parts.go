package completion

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jmuk/sylvan/pkg/chat/parts"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
)

func partToMessage(p parts.Part, l *slog.Logger) (openai.ChatCompletionMessageParamUnion, bool, error) {
	if p.Text != "" {
		return openai.UserMessage(p.Text), true, nil
	}
	if b := p.Audio; b != nil {
		var format string
		switch b.MimeType {
		case "audio/mpeg":
			format = "mp3"
		case "audio/wav":
			format = "wav"
		default:
			l.Warn("Unsupported mime type", "type", b.MimeType)
			return openai.ChatCompletionMessageParamUnion{}, false, nil
		}
		data := base64.StdEncoding.EncodeToString(b.Data)
		return openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
			openai.InputAudioContentPart(openai.ChatCompletionContentPartInputAudioInputAudioParam{
				Data:   data,
				Format: format,
			}),
		}), true, nil
	}
	if b := p.Image; b != nil {
		return openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
			openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
				URL:    b.DataURL(),
				Detail: "auto",
			}),
		}), true, nil
	}
	if b := p.File; b != nil {
		var filename param.Opt[string]
		if b.Filename != "" {
			filename = param.NewOpt(b.Filename)
		}
		return openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
			openai.FileContentPart(openai.ChatCompletionContentPartFileFileParam{
				FileData: param.NewOpt(base64.StdEncoding.EncodeToString(b.Data)),
				Filename: filename,
			}),
		}), true, nil
	}
	if ref := p.FileRef; ref != nil {
		l.Debug("Fileref not supported", "ref", ref)
		return openai.ChatCompletionMessageParamUnion{}, false, nil
	}
	if fr := p.FunctionResponse; fr != nil {
		var content []openai.ChatCompletionContentPartTextParam
		for i, frp := range fr.Parts {
			var text string
			if frp.Text != "" {
				text = frp.Text
			} else {
				enc, err := json.Marshal(frp)
				if err != nil {
					return openai.ChatCompletionMessageParamUnion{}, false, fmt.Errorf("part %d: %w", i, err)
				}
				text = string(enc)
			}
			content = append(content, openai.ChatCompletionContentPartTextParam{
				Text: text,
				Type: "text",
			})
		}
		msg := map[string]any{
			"success": fr.Error == nil,
		}
		if fr.Error == nil {
			msg["data"] = fr.Response
		} else {
			msg["error_message"] = fr.Error.Error()
		}
		enc, err := json.Marshal(msg)
		if err != nil {
			return openai.ChatCompletionMessageParamUnion{}, false, err
		}
		content = append(content, openai.ChatCompletionContentPartTextParam{
			Text: string(enc),
			Type: "text",
		})
		return openai.ToolMessage(content, fr.ID), true, nil
	}
	return openai.ChatCompletionMessageParamUnion{}, false, errors.New("not implemented")
}
