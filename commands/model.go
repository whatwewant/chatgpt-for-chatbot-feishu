package commands

import (
	"github.com/go-zoox/chatbot-feishu"
	chatgpt "github.com/go-zoox/chatgpt-client"
	"github.com/go-zoox/chatgpt-for-chatbot-feishu/config"
	"github.com/go-zoox/core-utils/fmt"
	"github.com/go-zoox/core-utils/strings"
	"github.com/go-zoox/feishu"
	feishuEvent "github.com/go-zoox/feishu/event"
)

func CreateModelCommand(
	feishuClient feishu.Client,
	chatgptClient chatgpt.Client,
	cfg *config.Config,
) *chatbot.Command {
	return &chatbot.Command{
		ArgsLength: 1,
		Handler: func(args []string, request *feishuEvent.EventRequest, reply func(content string, msgType ...string) error) error {
			if err := isAllowToDo(feishuClient, cfg, request, "model"); err != nil {
				return err
			}

			if len(args) == 0 || args[0] == "" {
				currentModel, err := chatgptClient.GetConversationModel(request.ChatID(), &chatgpt.ConversationConfig{
					MaxMessages: 100,
					Model:       cfg.OpenAIModel,
				})
				if err != nil {
					return fmt.Errorf("failed to get model by conversation(%s)", request.ChatID())
				}

				if err := replyText(reply, fmt.Sprintf("当前模型：%s", currentModel)); err != nil {
					return fmt.Errorf("failed to reply: %v", err)
				}

				return nil
			}

			model := args[0]
			if model == "" {
				return fmt.Errorf("model name is required (args: %s)", strings.Join(args, " "))
			}

			if err := chatgptClient.ChangeConversationModel(request.ChatID(), model, &chatgpt.ConversationConfig{
				MaxMessages: 50,
				Model:       cfg.OpenAIModel,
			}); err != nil {
				return fmt.Errorf("failed to set model(%s) for conversation(%s)", model, request.ChatID())
			}

			if err := replyText(reply, fmt.Sprintf("succeed to set model: %s", model)); err != nil {
				return fmt.Errorf("failed to reply: %v", err)
			}

			return nil
		},
	}
}
