package commands

import (
	"github.com/go-zoox/chatgpt-for-chatbot-feishu/config"
	"github.com/go-zoox/core-utils/fmt"
	"github.com/go-zoox/feishu"
	"github.com/go-zoox/feishu/contact/user"
	feishuEvent "github.com/go-zoox/feishu/event"
	mc "github.com/go-zoox/feishu/message/content"
)

func replyText(reply func(content string, msgType ...string) error, text string) error {
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

func replyImage(reply func(content string, msgType ...string) error, imageKey string) error {
	msgType, content, err := mc.
		NewContent().
		Image(&mc.ContentTypeImage{
			ImageKey: imageKey,
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

func isAllowToDo(feishuClient feishu.Client, cfg *config.Config, request *feishuEvent.EventRequest, command string) (reason error) {
	if cfg.AdminEmail != "" {
		eventSender, err := feishuClient.Contact().User().Retrieve(&user.RetrieveRequest{
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

func getUser(feishuClient feishu.Client, request *feishuEvent.EventRequest, cfg *config.Config) (*user.RetrieveResponse, error) {
	sender := request.Sender()

	if cfg.AdminEmail != "" {
		eventSender, err := feishuClient.Contact().User().Retrieve(&user.RetrieveRequest{
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
