package config

import (
	feishuBot "github.com/go-zoox/feishu/bot"
)

type Config struct {
	Port              int64
	APIPath           string
	OpenAIAPIKey      string
	OpenAIAPITimeout  int64
	AppID             string
	AppSecret         string
	EncryptKey        string
	VerificationToken string
	//
	ReportURL string
	//
	SiteURL string
	//
	OpenAIModel       string
	OpenAITemperature float64
	//
	FeishuBaseURI string
	//
	ConversationContext  string
	ConversationLanguage string
	//
	LogsDir   string
	LogsLevel string
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

	OpenAIAPIType         string
	OpenAIAzureResource   string
	OpenAIAzureDeployment string
	OpenAIAzureAPIVersion string

	// Custom Command with Service
	CustomCommand        string
	CustomCommandService string

	//
	Version string

	// @TODO State
	IsInService bool
	BotInfo     *feishuBot.GetBotInfoResponse
}
