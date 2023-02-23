package main

import (
	"strings"
	"time"

	"github.com/go-zoox/chalk"
	"github.com/go-zoox/chatbot-feishu"
	"github.com/go-zoox/core-utils/regexp"
	"github.com/go-zoox/debug"

	"github.com/go-zoox/core-utils/fmt"
	"github.com/go-zoox/feishu"
	mc "github.com/go-zoox/feishu/message/content"

	chatgpt "github.com/go-zoox/chatgpt-client"
	feishuBot "github.com/go-zoox/feishu/bot"
	feishuEvent "github.com/go-zoox/feishu/event"
	"github.com/go-zoox/logger"
	"github.com/go-zoox/retry"
)

type FeishuBotConfig struct {
	Port              int64
	APIPath           string
	ChatGPTAPIKey     string
	AppID             string
	AppSecret         string
	EncryptKey        string
	VerificationToken string
	//
	ReportURL string
	//
	SiteURL string
	//
	OpenAIModel string
	//
	FeishuBaseURI string
	//
	ChatGPTContext  string
	ChatGPTLanguage string
	//
	LogsDir string
}

func ServeFeishuBot(cfg *FeishuBotConfig) (err error) {
	logger.Infof("###### Settings START #######")
	logger.Infof("Serve at PORT: %d", cfg.Port)
	logger.Infof("Serve at API_PATH: %s", cfg.APIPath)
	logger.Infof("###### Settings END #######")

	logs := &Logs{
		Dir: cfg.LogsDir,
	}
	if err := logs.Setup(); err != nil {
		return fmt.Errorf("failed to setup logs: %v", err)
	}

	client, err := chatgpt.New(&chatgpt.Config{
		APIKey:               cfg.ChatGPTAPIKey,
		ConversationContext:  cfg.ChatGPTContext,
		ConversationLanguage: cfg.ChatGPTLanguage,
	})
	if err != nil {
		return fmt.Errorf("failed to create chatgpt client: %v", err)
	}

	bot := feishu.New(&feishu.Config{
		AppID:     cfg.AppID,
		AppSecret: cfg.AppSecret,
		BaseURI:   cfg.FeishuBaseURI,
	})
	var botInfo *feishuBot.GetBotInfoResponse

	tryToGetBotInfo := func() {
		for {
			if botInfo != nil {
				break
			}

			logger.Infof("Trying to get bot info ...")
			botInfo, err = bot.Bot().GetBotInfo()
			if err != nil {
				logger.Errorf("failed to get bot info: %v", err)
				return
			}

			logger.Infof("Bot Name: %s", botInfo.AppName)
			logger.Infof("Feishu Bot Online ...")
			time.Sleep(3 * time.Second)
		}
	}

	go func() {
		tryToGetBotInfo()
	}()

	if debug.IsDebugMode() {
		fmt.PrintJSON(map[string]interface{}{
			"cfg": cfg,
		})
	}

	if cfg.SiteURL != "" {
		logger.Infof("")
		logger.Infof("###### Feishu Configuration START #######")
		logger.Infof("# %s：%s", chalk.Red("飞书事件订阅请求地址"), chalk.Green(fmt.Sprintf("%s%s", cfg.SiteURL, cfg.APIPath)))
		logger.Infof("###### Feishu Configuration END #######")
		logger.Infof("")
	}

	feishuchatbot, err := chatbot.New(&chatbot.Config{
		Port:      cfg.Port,
		Path:      cfg.APIPath,
		AppID:     cfg.AppID,
		AppSecret: cfg.AppSecret,
	})
	if err != nil {
		return fmt.Errorf("failed to create feishu chatbot: %v", err)
	}

	feishuchatbot.OnCommand("ping", &chatbot.Command{
		Handler: func(args []string, request *feishuEvent.EventRequest, reply func(content string, msgType ...string) error) error {
			msgType, content, err := mc.
				NewContent().
				Post(&mc.ContentTypePost{
					ZhCN: &mc.ContentTypePostBody{
						Content: [][]mc.ContentTypePostBodyItem{
							{
								{
									Tag:      "text",
									UnEscape: true,
									Text:     "pong",
								},
							},
						},
					},
				}).
				Build()
			if err != nil {
				return fmt.Errorf("failed to build content: %v", err)
			}
			if err := reply(string(content), msgType); err != nil {
				return fmt.Errorf("failed to reply: %v", err)
			}

			return nil
		},
	})

	feishuchatbot.OnMessage(func(text string, request *feishuEvent.EventRequest, reply func(content string, msgType ...string) error) error {
		// fmt.PrintJSON(request)
		if botInfo == nil {
			logger.Infof("Trying to get bot info ...")
			botInfo, err = bot.Bot().GetBotInfo()
			if err != nil {
				return fmt.Errorf("failed to get bot info: %v", err)
			}
		}

		user := request.Sender().SenderID.UserID

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

		question = strings.TrimSpace(question)

		if question != "" {
			go func() {
				logger.Infof("%s 问：%s", user, question)
				var err error

				conversation, err := client.GetOrCreateConversation(request.ChatID(), &chatgpt.ConversationConfig{
					MaxMessages: 50,
					Model:       cfg.OpenAIModel,
				})
				if err != nil {
					logger.Errorf("failed to get or create conversation by ChatID %s", request.ChatID())
					return
				}

				if err := conversation.IsQuestionAsked(request.Header.EventID); err != nil {
					logger.Warnf("duplicated event(id: %s): %v", request.Header.EventID, err)
					return
				}

				var answer []byte
				err = retry.Retry(func() error {

					answer, err = conversation.Ask([]byte(question), &chatgpt.ConversationAskConfig{
						ID:   request.Header.EventID,
						User: user,
					})
					if err != nil {
						logger.Errorf("failed to request answer: %v", err)
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

				logger.Infof("ChatGPT 答：%s", answer)
				responseMessage := string(answer)
				if request.IsGroupChat() {
					responseMessage = fmt.Sprintf("%s\n-------------\n%s", question, answer)
				}

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
		}

		return nil
	})

	return feishuchatbot.Run()
}
