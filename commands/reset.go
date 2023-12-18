package commands

import (
	"github.com/go-zoox/chatbot-feishu"
	chatgpt "github.com/go-zoox/chatgpt-client"
	"github.com/go-zoox/chatgpt-for-chatbot-feishu/config"
	"github.com/go-zoox/core-utils/fmt"
	"github.com/go-zoox/feishu"
	feishuEvent "github.com/go-zoox/feishu/event"
)

func CreateResetCommand(
	feishuClient feishu.Client,
	chatgptClient chatgpt.Client,
	cfg *config.Config,
) *chatbot.Command {
	return &chatbot.Command{
		Handler: func(args []string, request *feishuEvent.EventRequest, reply chatbot.MessageReply) error {
			if request.IsGroupChat() {
				if err := isAllowToDo(feishuClient, cfg, request, "reset"); err != nil {
					return err
				}
			}

			if err := chatgptClient.ResetConversation(request.ChatID()); err != nil {
				return fmt.Errorf("failed to reset conversation(%s)", request.ChatID())
			}

			if err := replyText(reply, "succeed to reset"); err != nil {
				return fmt.Errorf("failed to reply: %v", err)
			}

			return nil
		},
	}
}
