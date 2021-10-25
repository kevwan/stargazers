package slack

import "github.com/slack-go/slack"

func Send(token, channel, message string) error {
	api := slack.New(token)
	attachment := slack.Attachment{
		Pretext: "Pretext",
		Text:    "Hello from GolangDocs!",
	}

	_, _, err := api.PostMessage(
		channel,
		slack.MsgOptionText("This is the main message", false),
		slack.MsgOptionAttachments(attachment),
		slack.MsgOptionAsUser(true),
	)
	return err
}
