package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-zoox/core-utils/fmt"

	"github.com/PullRequestInc/go-gpt3"
	"github.com/go-lark/lark"
	"github.com/go-zoox/core-utils/regexp"
	"github.com/go-zoox/logger"
	"github.com/go-zoox/retry"
	"github.com/go-zoox/zoox"
	"github.com/go-zoox/zoox/defaults"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/core/httpserverext"
	larkevent "github.com/larksuite/oapi-sdk-go/v3/event"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

type FeishuBotConfig struct {
	Port              int64
	ChatGPTAPIKey     string
	AppID             string
	AppSecret         string
	EncryptKey        string
	VerificationToken string
	//
	AsChallenge bool
}

func ServeFeishuBot(cfg *FeishuBotConfig) error {
	app := defaults.Application()

	client := gpt3.NewClient(cfg.ChatGPTAPIKey, gpt3.WithHTTPClient(&http.Client{
		Timeout: time.Duration(600 * time.Second),
	}))
	bot := lark.NewChatBot(cfg.AppID, cfg.AppSecret)
	_, _ = bot.GetTenantAccessTokenInternal(true)
	botInfo, err := bot.GetBotInfo()
	if err != nil {
		return fmt.Errorf("failed to get bot info: %v", err)
	}

	reply := func(chatID, answer string) error {
		msg := lark.NewMsgBuffer(lark.MsgPost)
		postContent := lark.NewPostBuilder().
			// Title("asdaads").
			TextTag(answer, 1, true).
			Render()
		om := msg.BindOpenChatID(chatID).Post(postContent).Build()

		return retry.Retry(func() error {
			resp, err := bot.PostMessage(om)
			if err != nil {
				logger.Errorf("failed to post message: %v", err)
				return fmt.Errorf("failed to request when reply: %v", err)
			}

			logger.Infof("robot response: %v", resp)

			//	Invalid access token for authorization. Please make a request with token attached
			// update the access token
			if resp.Code == 99991663 {
				_, _ = bot.GetTenantAccessTokenInternal(true)
				return fmt.Errorf("failed to reply: %s", resp.Msg)
			}

			return nil
		}, 3, 3*time.Second)
	}

	fmt.PrintJSON(map[string]interface{}{
		"cfg": cfg,
		"bot": botInfo.Bot,
	})

	// 注册消息处理器
	handler := dispatcher.NewEventDispatcher(cfg.VerificationToken, cfg.EncryptKey).
		OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
			// 处理消息 event，这里简单打印消息的内容
			fmt.Println(larkcore.Prettify(event))
			fmt.Println("OnP2MessageReceiveV1", event.RequestId())

			message := event.Event.Message.Content
			if message != nil {
				type Content struct {
					Text string `json:"text"`
				}
				var content Content
				if err := json.Unmarshal([]byte(*message), &content); err != nil {
					return err
				}

				textMessage := content.Text
				if textMessage == "" {
					return nil
				}

				fmt.Println("textMessage:", textMessage)

				chatID := *event.Event.Message.ChatId
				var question string

				// group chat
				if *event.Event.Message.ChatType == "group" {
					// @
					if ok := regexp.Match("^@_user_1", textMessage); ok {
						for _, metion := range event.Event.Message.Mentions {
							if *metion.Key == "@_user_1" && *metion.Id.OpenId == botInfo.Bot.OpenID {
								question = textMessage[len("@_user_1 "):]
								break
							}
						}
					} else if ok := regexp.Match("^/chatgpt\\s+", textMessage); ok {
						// command: /chatgpt
						question = textMessage[len("/chatgpt "):]
					}
				} else if *event.Event.Message.ChatType == "p2p" {
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
						var answer string
						var response *gpt3.CompletionResponse
						err = retry.Retry(func() error {
							response, err = client.CompletionWithEngine(context.Background(), gpt3.TextDavinci003Engine, gpt3.CompletionRequest{
								Prompt: []string{
									question,
								},
								MaxTokens:   gpt3.IntPtr(3000),
								Temperature: gpt3.Float32Ptr(0),
							})
							if err != nil {
								logger.Errorf("failed to request answer: %v", err)
								return fmt.Errorf("failed to request answer: %v", err)
							}

							answer = strings.TrimSpace(response.Choices[0].Text)

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
						responseMessage := answer
						if *event.Event.Message.ChatType == "group" {
							responseMessage = fmt.Sprintf("%s\n-------------\n%s", question, answer)
						}

						if err := reply(chatID, responseMessage); err != nil {
							return
						}
					}()
				}
			}

			return nil
		}).
		OnP2MessageReadV1(func(ctx context.Context, event *larkim.P2MessageReadV1) error {
			// 处理消息 event，这里简单打印消息的内容
			fmt.Println(larkcore.Prettify(event))
			fmt.Println("OnP2MessageReadV1", event.RequestId())
			return nil
		})

	// 注册 http 路由
	// http.HandleFunc("/webhook/event", httpserverext.NewEventHandlerFunc(handler, larkevent.WithLogLevel(larkcore.LogLevelDebug)))
	// http.HandleFunc("/bot/feishu", httpserverext.NewEventHandlerFunc(handler, larkevent.WithLogLevel(larkcore.LogLevelDebug)))
	app.Post("/bot/feishu", func(ctx *zoox.Context) {
		// body := map[string]interface{}{}
		// if err := ctx.BindJSON(&body); err != nil {
		// 	ctx.Fail(err, 500, "Internal Server Error")
		// 	return
		// }

		// fmt.PrintJSON(body)

		// ctx.Success(nil)
		// return
		if cfg.AsChallenge {
			type Challenge struct {
				Challenge string `json:"challenge"`
			}
			var c Challenge
			if err := ctx.BindJSON(&c); err != nil {
				ctx.Fail(err, 400000, "invalid challenge data")
				return
			}

			if c.Challenge == "" {
				ctx.Fail(fmt.Errorf("expect challenge, but got empty"), 400000, "expect challenge, but got empty")
				return
			}

			ctx.JSON(http.StatusOK, zoox.H{
				"challenge": c.Challenge,
			})
			return
		}

		httpserverext.NewEventHandlerFunc(handler, larkevent.WithLogLevel(larkcore.LogLevelDebug))(
			ctx.Writer,
			ctx.Request,
		)
	})

	// 启动 http 服务
	// return http.ListenAndServe(":8080", nil)
	return app.Run(fmt.Sprintf(":%d", cfg.Port))
}
