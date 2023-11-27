package commands

import (
	"net/http"

	"github.com/go-zoox/chatbot-feishu"
	chatgpt "github.com/go-zoox/chatgpt-client"
	"github.com/go-zoox/chatgpt-for-chatbot-feishu/config"
	"github.com/go-zoox/core-utils/fmt"
	"github.com/go-zoox/feishu"
	feishuEvent "github.com/go-zoox/feishu/event"
	"github.com/go-zoox/fetch"
	"github.com/go-zoox/logger"
)

func CreateCustomCommand(
	feishuClient feishu.Client,
	chatgptClient chatgpt.Client,
	cfg *config.Config,
) *chatbot.Command {
	return &chatbot.Command{
		ArgsLength: 1,
		Handler: func(args []string, request *feishuEvent.EventRequest, reply chatbot.MessageReply) error {
			if len(args) != 1 {
				return fmt.Errorf("invalid args: %v", args)
			}

			question := args[0]
			logger.Debugf("[custom command: %s, service: %s] question: %s", cfg.CustomCommand, cfg.CustomCommandService, question)

			response, err := fetch.Post(cfg.CustomCommandService, &fetch.Config{
				Headers: fetch.Headers{
					"Content-Type": "application/json",
					"Accept":       "application/json",
					"User-Agent":   fmt.Sprintf("go-zoox_fetch/%s chatgpt-for-chatbot-feishu/%s", fetch.Version, cfg.Version),
				},
				Body: map[string]interface{}{
					"question": args[0],
				},
			})
			if err != nil {
				logger.Errorf("failed to request from custom command service(%s)(1): %v", cfg.CustomCommandService, err)
				if err2 := replyText(reply, fmt.Sprintf("failed to interact with command service(err: %v)", err)); err2 != nil {
					return fmt.Errorf("failed to reply: %v", err)
				}

				return nil
			}

			if response.Status != http.StatusOK {
				logger.Errorf("failed to request from custom command service(%s)(2): %d", cfg.CustomCommandService, response.Status)
				if err := replyText(reply, fmt.Sprintf("failed to interact with command service (status: %d, response: %s)", response.Status, response.String())); err != nil {
					return fmt.Errorf("failed to reply: %v", err)
				}

				return nil
			}

			answer := response.Get("answer").String()
			if answer == "" {
				logger.Error("failed to request from custom command service(%s): empty answer (response: %s)", cfg.CustomCommandService, response.String())
				if err := replyText(reply, fmt.Sprintf("no answer found, unexpected response from custom command service(response: %s)", response.String())); err != nil {
					return fmt.Errorf("failed to reply: %v", err)
				}

				return nil
			}

			if err := replyText(reply, answer); err != nil {
				return fmt.Errorf("failed to reply: %v", err)
			}

			return nil
		},
	}
}
