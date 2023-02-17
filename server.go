package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-zoox/core-utils/regexp"

	"github.com/go-zoox/core-utils/fmt"
	"github.com/go-zoox/feishu"

	chatgpt "github.com/go-zoox/chatgpt-client"
	feishuEvent "github.com/go-zoox/feishu/event"
	"github.com/go-zoox/feishu/message"
	"github.com/go-zoox/logger"
	"github.com/go-zoox/retry"
	"github.com/go-zoox/zoox"
	"github.com/go-zoox/zoox/defaults"
)

type FeishuBotConfig struct {
	Port              int64
	ChatGPTAPIKey     string
	AppID             string
	AppSecret         string
	EncryptKey        string
	VerificationToken string
}

func ServeFeishuBot(cfg *FeishuBotConfig) error {
	app := defaults.Application()

	client, err := chatgpt.New(&chatgpt.Config{
		APIKey: cfg.ChatGPTAPIKey,
	})
	if err != nil {
		return fmt.Errorf("failed to create chatgpt client: %v", err)
	}

	// bot := lark.NewChatBot(cfg.AppID, cfg.AppSecret)

	bot := feishu.New(&feishu.Config{
		AppID:     cfg.AppID,
		AppSecret: cfg.AppSecret,
	})
	// _, _ = bot.GetTenantAccessTokenInternal(true)
	botInfo, err := bot.Bot().GetBotInfo()
	if err != nil {
		return fmt.Errorf("failed to get bot info: %v", err)
	}

	reply := func(chatID, answer string) error {
		// msg := lark.NewMsgBuffer(lark.MsgPost)
		// postContent := lark.NewPostBuilder().
		// 	// Title("asdaads").
		// 	TextTag(answer, 1, true).
		// 	Render()
		// om := msg.BindOpenChatID(chatID).Post(postContent).Build()

		content, err := json.Marshal(map[string]any{
			"zh_cn": map[string]any{
				"title": "",
				"content": [][]map[string]any{
					{
						{
							"tag":       "text",
							"un_escape": true,
							"text":      answer,
						},
					},
				},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to marshal content: %v", err)
		}

		//  "{\"zh_cn\":{\"title\":\"\",\"content\":[[{\"tag\":\"text\",\"un_escape\":true,\"text\":\"你好\\n-------------\\nNew\\n\\n你好！很高兴见到你！\",\"lines\":1}]]}}",
		fmt.Println("reply feishu content: ", string(content))

		return retry.Retry(func() error {
			// resp, err := bot.PostMessage(om)
			resp, err := bot.Message().Send(&message.SendRequest{
				ReceiveIDType: "chat_id",
				ReceiveID:     chatID,
				MsgType:       "post",
				Content:       string(content),
			})
			if err != nil {
				logger.Errorf("failed to post message: %v", err)
				return fmt.Errorf("failed to request when reply: %v", err)
			}

			logger.Infof("robot response: %v", resp)

			// //	Invalid access token for authorization. Please make a request with token attached
			// // update the access token
			// if resp.Code == 99991663 {
			// 	_, _ = bot.GetTenantAccessTokenInternal(true)
			// 	return fmt.Errorf("failed to reply: %s", resp.Msg)
			// }

			return nil
		}, 3, 3*time.Second)
	}

	fmt.PrintJSON(map[string]interface{}{
		"cfg": cfg,
		"bot": botInfo,
	})

	app.Post("/", func(ctx *zoox.Context) {
		var request feishuEvent.EventRequest
		if err := ctx.BindJSON(&request); err != nil {
			ctx.Fail(err, 500, "Internal Server Error")
			return
		}

		// fmt.PrintJSON(map[string]any{
		// 	"request":      request,
		// 	"is_challenge": request.IsChallenge(),
		// })

		if request.IsChallenge() {
			// type Challenge struct {
			// 	Challenge string `json:"challenge"`
			// }
			// var c Challenge
			// if err := ctx.BindJSON(&c); err != nil {
			// 	ctx.Fail(err, 400000, "invalid challenge data")
			// 	return
			// }

			if request.Challenge == "" {
				ctx.Fail(fmt.Errorf("expect challenge, but got empty"), 400000, "expect challenge, but got empty")
				return
			}

			ctx.JSON(http.StatusOK, zoox.H{
				"challenge": request.Challenge,
			})

			return
		}

		event := bot.Event(&request)
		go event.OnChatReceiveMessage(func(contentString string, request *feishuEvent.EventRequest, replyx func(content string) error) error {
			fmt.Println("onChatReceiveMessage: ....")

			if contentString != "" {
				type Content struct {
					Text string `json:"text"`
				}
				var content Content
				if err := json.Unmarshal([]byte(contentString), &content); err != nil {
					return err
				}

				textMessage := content.Text
				if textMessage == "" {
					return nil
				}

				fmt.Println("textMessage:", textMessage)

				chatID := request.ChatID()
				var question string
				fmt.Println("chatID:", chatID)

				// group chat
				if request.IsGroupChat() {
					// @
					if ok := regexp.Match("^@_user_1", textMessage); ok {
						for _, metion := range request.Event.Message.Mentions {
							if metion.Key == "@_user_1" && metion.ID.OpenID == botInfo.OpenID {
								question = textMessage[len("@_user_1 "):]
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

				if question != "" {
					fmt.Println("question:", question)

					go func() {
						logger.Infof("问题：%s", question)
						// if err := reply(chatID, "我想想 ..."); err != nil {
						// 	return
						// }

						var err error

						var answer []byte
						err = retry.Retry(func() error {
							conversation, err := client.GetOrCreateConversation(chatID, &chatgpt.ConversationConfig{})
							if err != nil {
								return fmt.Errorf("failed to get or create conversation by ChatID %s", chatID)
							}

							answer, err = conversation.Ask([]byte(question))
							if err != nil {
								logger.Errorf("failed to request answer: %v", err)
								return fmt.Errorf("failed to request answer: %v", err)
							}

							return nil
						}, 5, 3*time.Second)
						if err != nil {
							logger.Errorf("failed to get answer: %v", err)
							if err := reply(chatID, "ChatGPT 繁忙，请稍后重试"); err != nil {
								return
							}
							return
						}

						logger.Infof("回答：%s", answer)
						responseMessage := string(answer)
						if request.IsGroupChat() {
							responseMessage = fmt.Sprintf("%s\n-------------\n%s", question, answer)
						}

						if err := reply(chatID, responseMessage); err != nil {
							return
						}
					}()
				}
			}

			return nil
		})

		ctx.Success(nil)
	})

	// 启动 http 服务
	// return http.ListenAndServe(":8080", nil)
	return app.Run(fmt.Sprintf(":%d", cfg.Port))
}
