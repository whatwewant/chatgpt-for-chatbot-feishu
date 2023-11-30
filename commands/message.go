package commands

import (
	"time"

	"github.com/go-zoox/core-utils/regexp"

	"github.com/go-zoox/chatbot-feishu"
	chatgpt "github.com/go-zoox/chatgpt-client"
	"github.com/go-zoox/chatgpt-for-chatbot-feishu/config"
	"github.com/go-zoox/core-utils/fmt"
	"github.com/go-zoox/core-utils/strings"
	"github.com/go-zoox/feishu"
	feishuEvent "github.com/go-zoox/feishu/event"
	mc "github.com/go-zoox/feishu/message/content"
	"github.com/go-zoox/logger"
	"github.com/go-zoox/retry"
)

func CreateMessageCommand(
	feishuClient feishu.Client,
	chatgptClient chatgpt.Client,
	cfg *config.Config,
) *chatbot.Command {
	return &chatbot.Command{
		Handler: func(args []string, request *feishuEvent.EventRequest, reply func(content string, msgType ...string) error) (err error) {
			text := strings.Join(args, " ")

			// fmt.PrintJSON(request)
			if cfg.BotInfo == nil {
				logger.Infof("Trying to get bot info ...")
				cfg.BotInfo, err = feishuClient.Bot().GetBotInfo()
				if err != nil {
					return fmt.Errorf("failed to get bot info: %v", err)
				}
			}

			user, err := getUser(feishuClient, request, cfg)
			if err != nil {
				return fmt.Errorf("failed to get user: %v", err)
			}

			textMessage := strings.TrimSpace(text)
			if textMessage == "" {
				return nil
			}

			var question string
			// group chat
			if request.IsGroupChat() {
				// @
				if ok := regexp.Match("^@_user_1", textMessage); ok {
					for _, metion := range request.Event.Message.Mentions {
						if metion.Key == "@_user_1" && metion.ID.OpenID == cfg.BotInfo.OpenID {
							question = textMessage[len("@_user_1"):]
							question = strings.TrimSpace(question)
							break
						}
					}
				} else if ok := regexp.Match("^/chatgpt\\s+", textMessage); ok {
					// command: /chatgpt
					question = textMessage[len("/chatgpt "):]
				}
			} else if request.IsP2pChat() {
				question = textMessage
			}

			question = strings.TrimSpace(question)
			if question == "" {
				logger.Infof("ignore empty question message")
				return nil
			}

			// @TODO 离线服务
			if !cfg.IsInService {
				return replyText(reply, cfg.OfflineMessage)
			}

			go func() {
				logger.Debugf("%s 问 ChatGPT：%s", user.User.Name, question)

				var err error

				conversation, err := chatgptClient.GetOrCreateConversation(request.ChatID(), &chatgpt.ConversationConfig{
					MaxMessages: 50,
					Model:       cfg.OpenAIModel,
					Temperature: cfg.OpenAITemperature,
				})
				if err != nil {
					logger.Errorf("failed to get or create conversation by ChatID %s", request.ChatID())
					return
				}

				if err := conversation.IsQuestionAsked(request.Event.Message.MessageID); err != nil {
					logger.Warnf("duplicated event(id: %s): %v", request.Event.Message.MessageID, err)
					return
				}

				var answer []byte
				err = retry.Retry(func() error {

					answer, err = conversation.Ask([]byte(question), &chatgpt.ConversationAskConfig{
						ID:   request.Event.Message.MessageID,
						User: user.User.Name,
					})
					if err != nil {
						logger.Errorf("failed to request answer: %v", err)
						replyText(reply, fmt.Sprintf("服务异常：%s", err.Error()))
						return fmt.Errorf("failed to request answer: %v", err)
					}

					return nil
				}, 5, 3*time.Second)
				if err != nil {
					logger.Errorf("failed to get answer: %v", err)
					msgType, content, err := mc.
						NewContent().
						Text(&mc.ContentTypeText{
							Text: "ChatGPT 繁忙，请稍后重试",
						}).
						Build()
					if err != nil {
						logger.Errorf("failed to build content: %v", err)
						return
					}
					if err := reply(string(content), msgType); err != nil {
						return
					}
					return
				}

				logger.Debugf("ChatGPT 答 %s：%s", user.User.Name, answer)

				responseMessage := string(answer)
				// if request.IsGroupChat() {
				// 	responseMessage = fmt.Sprintf("%s\n-------------\n%s", question, answer)
				// }

				msgType, content, err := mc.
					NewContent().
					Post(&mc.ContentTypePost{
						ZhCN: &mc.ContentTypePostBody{
							Content: [][]mc.ContentTypePostBodyItem{
								{
									{
										Tag:      "text",
										UnEscape: true,
										Text:     responseMessage,
									},
								},
							},
						},
					}).
					Build()
				if err != nil {
					logger.Errorf("failed to build content: %v", err)
					return
				}
				if err := reply(string(content), msgType); err != nil {
					logger.Errorf("failed to reply: %v", err)
					return
				}
			}()

			return nil
		},
	}
}
