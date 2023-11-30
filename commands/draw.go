package commands

import (
	"github.com/go-zoox/chatbot-feishu"
	chatgpt "github.com/go-zoox/chatgpt-client"
	"github.com/go-zoox/core-utils/fmt"
	"github.com/go-zoox/core-utils/strings"
	"github.com/go-zoox/feishu"
	feishuEvent "github.com/go-zoox/feishu/event"
	feishuImage "github.com/go-zoox/feishu/image"
	"github.com/go-zoox/fetch"
	"github.com/go-zoox/fs"
	"github.com/go-zoox/logger"
	openaiclient "github.com/go-zoox/openai-client"
)

func CreateDrawCommand(
	feishuClient feishu.Client,
	chatgptClient chatgpt.Client,
) *chatbot.Command {
	return &chatbot.Command{
		Handler: func(args []string, request *feishuEvent.EventRequest, reply chatbot.MessageReply) error {
			prompt := strings.Join(args, " ")
			if prompt == "" {
				return replyText(reply, fmt.Sprintf("prompt is required (args: %s)", strings.Join(args, " ")))
			}

			logger.Infof("[draw]: %v", prompt)
			replyText(reply, "创作中，请稍等 ...")

			logger.Infof("[draw]: request image generation ...")
			response, err := chatgptClient.ImageGeneration(&openaiclient.ImageGenerationRequest{
				Prompt: prompt,
			})
			if err != nil {
				return replyText(reply, fmt.Sprintf("failed to request image generation: %v", err))
			}

			for _, image := range response.Data {
				tmpFilePath := fs.TmpFilePath()

				logger.Infof("[draw] download image from chatgpt: %v", image.URL)
				_, err := fetch.Download(image.URL, tmpFilePath, &fetch.Config{})
				if err != nil {
					return replyText(reply, fmt.Sprintf("failed to download image: %v", err))
				}

				tmpFile, err := fs.Open(tmpFilePath)
				if err != nil {
					return replyText(reply, fmt.Sprintf("failed to open image: %v", err))
				}

				logger.Infof("[draw] upload image to feishu ...")
				response, err := feishuClient.Image().Upload(&feishuImage.UploadRequest{
					ImageType: "message",
					Image:     tmpFile,
				})
				if err != nil {
					return replyText(reply, fmt.Sprintf("failed to upload image: %v", err))
				}

				logger.Infof("[draw] reply image to feishu: %v", response.ImageKey)
				replyImage(reply, response.ImageKey)
			}

			return nil
		},
	}
}
