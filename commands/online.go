package commands

import (
	"github.com/go-zoox/chatbot-feishu"
	chatgpt "github.com/go-zoox/chatgpt-client"
	"github.com/go-zoox/chatgpt-for-chatbot-feishu/config"
	"github.com/go-zoox/core-utils/fmt"
	"github.com/go-zoox/feishu"
	feishuEvent "github.com/go-zoox/feishu/event"
)

func CreateOnlineCommand(
	feishuClient feishu.Client,
	chatgptClient chatgpt.Client,
	cfg *config.Config,
) *chatbot.Command {
	return &chatbot.Command{
		Handler: func(args []string, request *feishuEvent.EventRequest, reply func(content string, msgType ...string) error) error {
			if err := isAllowToDo(feishuClient, cfg, request, "online"); err != nil {
				return err
			}

			cfg.IsInService = true

			if err := replyText(reply, "succeed to online"); err != nil {
				return fmt.Errorf("failed to reply: %v", err)
			}

			return nil
		},
	}
}
