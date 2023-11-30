package main

import (
	"github.com/go-zoox/chatgpt-for-chatbot-feishu/config"
	"github.com/go-zoox/cli"
)

func main() {
	app := cli.NewSingleProgram(&cli.SingleProgramConfig{
		Name:    "chatgpt-for-chatbot-feishu",
		Usage:   "chatgpt-for-chatbot-feishu is a portable chatgpt server",
		Version: Version,
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "port",
				Usage:   "server port",
				Aliases: []string{"p"},
				EnvVars: []string{"PORT"},
				Value:   8080,
			},
			&cli.StringFlag{
				Name:    "api-path",
				Usage:   "custom api path, default: /",
				EnvVars: []string{"API_PATH"},
				Value:   "/",
			},
			&cli.StringFlag{
				Name:    "openai-api-key",
				Usage:   "OpenAI API Key",
				EnvVars: []string{"OPENAI_API_KEY"},
				// Required: true,
			},
			&cli.Int64Flag{
				Name:    "openai-api-timeout",
				Usage:   "OpenAI API Timeout, unit: second, default: 300",
				EnvVars: []string{"OPENAI_API_TIMEOUT"},
				Value:   300,
			},
			&cli.StringFlag{
				Name:    "openai-api-server",
				Usage:   "OpenAI API Server",
				EnvVars: []string{"OPENAI_API_SERVER"},
			},
			&cli.StringFlag{
				Name:    "openai-api-type",
				Usage:   "OpenAI API Type",
				EnvVars: []string{"OPENAI_API_TYPE"},
			},
			&cli.StringFlag{
				Name:    "openai-azure-resource",
				Usage:   "Azure OpenAI Service Resource",
				EnvVars: []string{"OPENAI_AZURE_RESOURCE"},
			},
			&cli.StringFlag{
				Name:    "openai-azure-deployment",
				Usage:   "Azure OpenAI Service Deployment",
				EnvVars: []string{"OPENAI_AZURE_DEPLOYMENT"},
			},
			&cli.StringFlag{
				Name:    "openai-azure-api-version",
				Usage:   "Azure OpenAI Service API Version",
				EnvVars: []string{"OPENAI_AZURE_API_VERSION"},
			},
			&cli.StringFlag{
				Name:     "app-id",
				Usage:    "Feishu App ID",
				EnvVars:  []string{"APP_ID"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     "app-secret",
				Usage:    "Feishu App SECRET",
				EnvVars:  []string{"APP_SECRET"},
				Required: true,
			},
			&cli.StringFlag{
				Name:    "encrypt-key",
				Usage:   "enable encryption if you need",
				EnvVars: []string{"ENCRYPT_KEY"},
			},
			&cli.StringFlag{
				Name:    "verification-token",
				Usage:   "enable token verification if you need",
				EnvVars: []string{"VERIFICATION_TOKEN"},
			},
			&cli.StringFlag{
				Name:    "report-url",
				Usage:   "Set error report url",
				EnvVars: []string{"REPORT_URL"},
			},
			&cli.StringFlag{
				Name:    "site-url",
				Usage:   "The Site URL",
				EnvVars: []string{"SITE_URL"},
			},
			&cli.StringFlag{
				Name:    "openai-model",
				Usage:   "Custom open ai model",
				EnvVars: []string{"OPENAI_MODEL"},
			},
			&cli.Float64Flag{
				Name:    "openai-temperature",
				Usage:   "Custom open ai temperature",
				EnvVars: []string{"OPENAI_TEMPERATURE"},
				Value:   0.3,
			},
			&cli.StringFlag{
				Name:    "feishu-base-uri",
				Usage:   "Custom feishu base uri for selfhosted Feishu",
				EnvVars: []string{"FEISHU_BASE_URI"},
			},
			&cli.StringFlag{
				Name:    "conversation-context",
				Usage:   "Custom chatgpt conversation context",
				EnvVars: []string{"CONVERSATION_CONTEXT"},
			},
			&cli.StringFlag{
				Name:    "conversation-language",
				Usage:   "Custom chatgpt conversation lanuage",
				EnvVars: []string{"CONVERSATION_LANGUAGE"},
			},
			&cli.StringFlag{
				Name:    "logs-dir",
				Usage:   "The logs dir for save logs",
				EnvVars: []string{"LOGS_DIR"},
				Value:   "/tmp/chatgpt-for-chatbot-feishu",
			},
			&cli.StringFlag{
				Name:    "logs-level",
				Usage:   "The logs level",
				EnvVars: []string{"LOGS_LEVEL", "LOG_LEVEL"},
				Value:   "INFO",
			},
			&cli.StringFlag{
				Name:    "offline-message",
				Usage:   "The message to use for offline status",
				EnvVars: []string{"OFFLINE_MESSAGE"},
				Value:   "robot is offline",
			},
			&cli.StringFlag{
				Name:    "admin-email",
				Usage:   "Sets the admin with admin email, who can run commands",
				EnvVars: []string{"ADMIN_EMAIL"},
			},
			&cli.StringFlag{
				Name:    "bot-name",
				Usage:   "Sets the bot name, default: ChatGPT",
				EnvVars: []string{"BOT_NAME"},
			},
			&cli.StringFlag{
				Name:    "proxy",
				Usage:   "Sets the request proxy",
				EnvVars: []string{"PROXY", "HTTPS_PROXY"},
			},
			&cli.StringFlag{
				Name:    "custom-command",
				Usage:   "Custom command, such as: doc => trigger /doc",
				EnvVars: []string{"CUSTOM_COMMAND"},
			},
			&cli.StringFlag{
				Name:    "custom-command-service",
				Usage:   "Custom command service, such as: https://example.com/api/doc",
				EnvVars: []string{"CUSTOM_COMMAND_SERVICE"},
			},
		},
	})

	app.Command(func(ctx *cli.Context) (err error) {
		return ServeFeishuBot(&config.Config{
			LogsDir:               ctx.String("logs-dir"),
			LogsLevel:             ctx.String("logs-level"),
			Port:                  ctx.Int64("port"),
			APIPath:               ctx.String("api-path"),
			OpenAIAPIKey:          ctx.String("openai-api-key"),
			OpenAIAPITimeout:      ctx.Int64("openai-api-timeout"),
			AppID:                 ctx.String("app-id"),
			AppSecret:             ctx.String("app-secret"),
			EncryptKey:            ctx.String("encrypt-key"),
			VerificationToken:     ctx.String("verification-token"),
			ReportURL:             ctx.String("report-url"),
			SiteURL:               ctx.String("site-url"),
			OpenAIModel:           ctx.String("openai-model"),
			OpenAITemperature:     ctx.Float64("openai-temperature"),
			FeishuBaseURI:         ctx.String("feishu-base-uri"),
			ConversationContext:   ctx.String("conversation-context"),
			ConversationLanguage:  ctx.String("conversation-language"),
			OfflineMessage:        ctx.String("offline-message"),
			AdminEmail:            ctx.String("admin-email"),
			BotName:               ctx.String("bot-name"),
			Proxy:                 ctx.String("proxy"),
			OpenAIAPIServer:       ctx.String("openai-api-server"),
			OpenAIAPIType:         ctx.String("openai-api-type"),
			OpenAIAzureResource:   ctx.String("openai-azure-resource"),
			OpenAIAzureDeployment: ctx.String("openai-azure-deployment"),
			OpenAIAzureAPIVersion: ctx.String("openai-azure-api-version"),
			CustomCommand:         ctx.String("custom-command"),
			CustomCommandService:  ctx.String("custom-command-service"),
		})
	})

	app.Run()
}
