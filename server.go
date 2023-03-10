package main

import (
	"net/http"
	"time"

	"github.com/go-zoox/chalk"
	"github.com/go-zoox/chatbot-feishu"
	"github.com/go-zoox/core-utils/regexp"
	"github.com/go-zoox/core-utils/strings"
	openaiclient "github.com/go-zoox/openai-client"
	"github.com/go-zoox/proxy"
	"github.com/go-zoox/proxy/utils/rewriter"
	"github.com/go-zoox/zoox"
	"github.com/go-zoox/zoox/defaults"
	"github.com/go-zoox/zoox/middleware"

	"github.com/go-zoox/core-utils/fmt"
	"github.com/go-zoox/feishu"
	"github.com/go-zoox/feishu/contact/user"
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
	OpenAIAPIKey      string
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
	//
	OfflineMessage string
	//
	AdminEmail string
	//
	BotName string

	// Proxy sets the request proxy.
	// support http, https, socks5
	// example:
	//   http://127.0.0.1:17890
	//   https://127.0.0.1:17890
	//   socks5://127.0.0.1:17890
	Proxy string

	OpenAIAPIServer string

	// ProxyOpenAIAPIPath proxys the OpenAPI API (https://api.openai.com)
	ProxyOpenAIAPIPath string
	// ProxyOpenAIAPIToken limits auth  with Bearer Token
	ProxyOpenAIAPIToken string
}

func ServeFeishuBot(cfg *FeishuBotConfig) (err error) {
	if cfg.OfflineMessage == "" {
		cfg.OfflineMessage = "robot is offline"
	}

	logger.Infof("###### Settings START #######")
	logger.Infof("Serve Version: %s", Version)
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
		APIKey:               cfg.OpenAIAPIKey,
		APIServer:            cfg.OpenAIAPIServer,
		ConversationContext:  cfg.ChatGPTContext,
		ConversationLanguage: cfg.ChatGPTLanguage,
		ChatGPTName:          cfg.BotName,
		Proxy:                cfg.Proxy,
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

	isInService := true

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

	isAllowToDo := func(request *feishuEvent.EventRequest, command string) (reason error) {
		if cfg.AdminEmail != "" {
			eventSender, err := bot.Contact().User().Retrieve(&user.RetrieveRequest{
				UserIDType: "open_id",
				UserID:     request.Sender().SenderID.OpenID,
			})
			if err != nil {
				return fmt.Errorf("failed to retrieve user with openid(%s): %v", request.Sender().SenderID.OpenID, err)
			}

			if eventSender.User.EnterpriseEmail != cfg.AdminEmail && eventSender.User.Email != cfg.AdminEmail {
				return fmt.Errorf("user(%s) is not allow to do action: %s", eventSender.User.Name, command)
			}

			return nil
		}

		return fmt.Errorf("admin email is not set, not allow to do action: %s", command)
	}

	getUser := func(request *feishuEvent.EventRequest) (*user.RetrieveResponse, error) {
		sender := request.Sender()

		if cfg.AdminEmail != "" {
			eventSender, err := bot.Contact().User().Retrieve(&user.RetrieveRequest{
				UserIDType: "open_id",
				UserID:     sender.SenderID.OpenID,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve user with openid(%s): %v", sender.SenderID.OpenID, err)
			}

			return eventSender, nil
		}

		return &user.RetrieveResponse{
			User: user.UserEntity{
				Name:    sender.SenderID.UserID,
				OpenID:  sender.SenderID.OpenID,
				UnionID: sender.SenderID.UnionID,
				UserID:  sender.SenderID.UserID,
			},
		}, nil
	}

	go func() {
		tryToGetBotInfo()
	}()

	// if debug.IsDebugMode() {
	fmt.PrintJSON(map[string]interface{}{
		"version": Version,
		"cfg":     cfg,
	})
	// }

	if cfg.SiteURL != "" {
		logger.Infof("")
		logger.Infof("###### Feishu Configuration START #######")
		logger.Infof("# %s：%s", chalk.Red("飞书事件订阅请求地址"), chalk.Green(fmt.Sprintf("%s%s", cfg.SiteURL, cfg.APIPath)))
		logger.Infof("###### Feishu Configuration END #######")
		logger.Infof("")
	}

	feishuchatbot, err := chatbot.New(&chatbot.Config{
		Port:              cfg.Port,
		Path:              cfg.APIPath,
		AppID:             cfg.AppID,
		AppSecret:         cfg.AppSecret,
		VerificationToken: cfg.VerificationToken,
		EncryptKey:        cfg.EncryptKey,
	})
	if err != nil {
		return fmt.Errorf("failed to create feishu chatbot: %v", err)
	}

	replyText := func(reply func(content string, msgType ...string) error, text string) error {
		msgType, content, err := mc.
			NewContent().
			Post(&mc.ContentTypePost{
				ZhCN: &mc.ContentTypePostBody{
					Content: [][]mc.ContentTypePostBodyItem{
						{
							{
								Tag:      "text",
								UnEscape: true,
								Text:     text,
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
	}

	feishuchatbot.OnCommand("ping", &chatbot.Command{
		Handler: func(args []string, request *feishuEvent.EventRequest, reply func(content string, msgType ...string) error) error {
			if err := replyText(reply, "pong"); err != nil {
				return fmt.Errorf("failed to reply: %v", err)
			}

			return nil
		},
	})

	feishuchatbot.OnCommand("offline", &chatbot.Command{
		Handler: func(args []string, request *feishuEvent.EventRequest, reply func(content string, msgType ...string) error) error {
			if err := isAllowToDo(request, "online"); err != nil {
				return err
			}

			isInService = false

			if err := replyText(reply, "succeeed to offline"); err != nil {
				return fmt.Errorf("failed to reply: %v", err)
			}

			return nil
		},
	})

	feishuchatbot.OnCommand("online", &chatbot.Command{
		Handler: func(args []string, request *feishuEvent.EventRequest, reply func(content string, msgType ...string) error) error {
			if err := isAllowToDo(request, "online"); err != nil {
				return err
			}

			isInService = true

			if err := replyText(reply, "succeeed to online"); err != nil {
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

		user, err := getUser(request)
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
					if metion.Key == "@_user_1" && metion.ID.OpenID == botInfo.OpenID {
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
		if !isInService {
			return replyText(reply, cfg.OfflineMessage)
		}

		go func() {
			logger.Infof("%s 问 ChatGPT：%s", user.User.Name, question)
			var err error

			conversation, err := client.GetOrCreateConversation(request.ChatID(), &chatgpt.ConversationConfig{
				MaxMessages: 50,
				Model:       cfg.OpenAIModel,
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

			logger.Infof("ChatGPT 答 %s：%s", user.User.Name, answer)
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

		return nil
	})

	// return feishuchatbot.Run()

	return run(
		feishuchatbot,
		cfg.Port,
		cfg.APIPath,
		cfg.OpenAIAPIKey,
		cfg.OpenAIAPIServer,
		cfg.ProxyOpenAIAPIPath,
		cfg.ProxyOpenAIAPIToken,
	)
}

func run(
	chatbot chatbot.ChatBot,
	port int64,
	path string,
	OpenAIAPIKey string,
	OpenAIAPIServer string,
	ProxyOpenAIAPIPath string,
	ProxyOpenAIAPIToken string,
) error {
	if OpenAIAPIServer == "" {
		OpenAIAPIServer = openaiclient.DefaultAPIServer
	}

	app := defaults.Application()

	app.Post(path, chatbot.Handler())

	app.Get(path, func(ctx *zoox.Context) {
		ctx.String(200, "OK")
	})

	if ProxyOpenAIAPIPath != "" {
		if ProxyOpenAIAPIToken == "" {
			return fmt.Errorf("env ProxyOpenAIAPIToken is required when ProxyOpenAIAPIPath sets")
		}

		app.Group(ProxyOpenAIAPIPath, func(group *zoox.RouterGroup) {
			group.Use(middleware.BearerToken(strings.Split(ProxyOpenAIAPIToken, ",")))

			group.Proxy("/", OpenAIAPIServer, &proxy.SingleTargetConfig{
				RequestHeaders: http.Header{
					"Authorization": []string{fmt.Sprintf("Bearer %s", OpenAIAPIKey)},
					"User-Agent":    []string{fmt.Sprintf("GoZoox/ChatGPT-for-ChatBot-Feishu@%s", Version)},
				},
				Rewrites: rewriter.Rewriters{
					{
						From: fmt.Sprintf("^%s/(.*)$", ProxyOpenAIAPIPath),
						To:   "/$1",
					},
				},
			})
		})
	}

	return app.Run(fmt.Sprintf(":%d", port))
}
