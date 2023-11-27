package main

import (
	"time"

	"github.com/go-zoox/chalk"
	"github.com/go-zoox/chatbot-feishu"
	"github.com/go-zoox/chatgpt-for-chatbot-feishu/commands"
	"github.com/go-zoox/chatgpt-for-chatbot-feishu/config"
	openaiclient "github.com/go-zoox/openai-client"
	"github.com/go-zoox/zoox"
	"github.com/go-zoox/zoox/defaults"

	"github.com/go-zoox/core-utils/fmt"
	"github.com/go-zoox/feishu"

	chatgpt "github.com/go-zoox/chatgpt-client"
	feishuEvent "github.com/go-zoox/feishu/event"
	"github.com/go-zoox/logger"
)

func ServeFeishuBot(cfg *config.Config) (err error) {
	if cfg.OfflineMessage == "" {
		cfg.OfflineMessage = "robot is offline"
	}

	if cfg.OpenAIModel == "" {
		cfg.OpenAIModel = openaiclient.ModelGPT_4
	}

	logger.Infof("###### Settings START #######")
	logger.Infof("Serve Version: %s", Version)
	logger.Infof("Serve at PORT: %d", cfg.Port)
	logger.Infof("Serve at API_PATH: %s", cfg.APIPath)
	logger.Infof("###### Settings END #######")

	logs := &Logs{
		Dir:   cfg.LogsDir,
		Level: cfg.LogsLevel,
	}
	if err := logs.Setup(); err != nil {
		return fmt.Errorf("failed to setup logs: %v", err)
	}

	chatgptClient, err := chatgpt.New(&chatgpt.Config{
		APIKey:               cfg.OpenAIAPIKey,
		APIServer:            cfg.OpenAIAPIServer,
		APIType:              cfg.OpenAIAPIType,
		AzureResource:        cfg.OpenAIAzureResource,
		AzureDeployment:      cfg.OpenAIAzureDeployment,
		AzureAPIVersion:      cfg.OpenAIAzureAPIVersion,
		ConversationContext:  cfg.ConversationContext,
		ConversationLanguage: cfg.ConversationLanguage,
		ChatGPTName:          cfg.BotName,
		Proxy:                cfg.Proxy,
		Timeout:              time.Duration(cfg.OpenAIAPITimeout) * time.Second,
	})
	if err != nil {
		return fmt.Errorf("failed to create chatgpt client: %v", err)
	}

	feishuClient := feishu.New(&feishu.Config{
		AppID:     cfg.AppID,
		AppSecret: cfg.AppSecret,
		BaseURI:   cfg.FeishuBaseURI,
	})

	cfg.Version = Version
	cfg.IsInService = true

	tryToGetBotInfo := func() {
		for {
			if cfg.BotInfo != nil {
				break
			}

			logger.Infof("Trying to get bot info ...")
			cfg.BotInfo, err = feishuClient.Bot().GetBotInfo()
			if err != nil {
				logger.Errorf("failed to get bot info: %v", err)
				return
			}

			logger.Infof("Bot Name: %s", cfg.BotInfo.AppName)
			logger.Infof("Feishu Bot Online ...")
			time.Sleep(3 * time.Second)
		}
	}

	go func() {
		tryToGetBotInfo()
	}()

	fmt.PrintJSON(map[string]interface{}{
		"version": Version,
		"cfg":     cfg,
	})

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

	feishuchatbot.OnCommand("ping", commands.CreatePingCommand(feishuClient, chatgptClient))

	feishuchatbot.OnCommand("offline", commands.CreateOfflineCommand(feishuClient, chatgptClient, cfg))

	feishuchatbot.OnCommand("online", commands.CreateOnlineCommand(feishuClient, chatgptClient, cfg))

	feishuchatbot.OnCommand("model", commands.CreateModelCommand(feishuClient, chatgptClient, cfg))

	feishuchatbot.OnCommand("reset", commands.CreateResetCommand(feishuClient, chatgptClient, cfg))

	feishuchatbot.OnCommand("message", commands.CreateMessageCommand(feishuClient, chatgptClient, cfg))
	feishuchatbot.OnCommand("问答", commands.CreateMessageCommand(feishuClient, chatgptClient, cfg))

	feishuchatbot.OnCommand("draw", commands.CreateDrawCommand(feishuClient, chatgptClient))
	feishuchatbot.OnCommand("画图", commands.CreateDrawCommand(feishuClient, chatgptClient))

	if cfg.CustomCommand != "" && cfg.CustomCommandService != "" {
		feishuchatbot.OnCommand(cfg.CustomCommand, commands.CreateCustomCommand(feishuClient, chatgptClient, cfg))
	}

	feishuchatbot.OnMessage(func(text string, request *feishuEvent.EventRequest, reply func(content string, msgType ...string) error) error {
		return commands.
			CreateMessageCommand(feishuClient, chatgptClient, cfg).
			Handler([]string{text}, request, reply)
	})

	// return feishuchatbot.Run()

	return run(
		feishuchatbot,
		cfg.Port,
		cfg.APIPath,
		cfg.OpenAIAPIKey,
		cfg.OpenAIAPIServer,
		cfg.OpenAIAPIType,
		cfg.OpenAIAzureResource,
		cfg.OpenAIAzureDeployment,
		cfg.OpenAIAzureAPIVersion,
	)
}

func run(
	chatbot chatbot.ChatBot,
	port int64,
	path,
	OpenAIAPIKey,
	OpenAIAPIServer,
	OpenAIAPIType,
	OpenAIAzureResource,
	OpenAIAzureDeployment,
	OpenAIAzureAPIVersion string,
) error {
	if OpenAIAPIServer == "" {
		OpenAIAPIServer = openaiclient.DefaultAPIServer
	}

	app := defaults.Application()

	app.Post(path, chatbot.Handler())

	app.Get(path, func(ctx *zoox.Context) {
		ctx.String(200, "OK")
	})

	return app.Run(fmt.Sprintf(":%d", port))
}
