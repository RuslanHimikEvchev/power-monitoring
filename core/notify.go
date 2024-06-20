package core

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go-meshtastic-monitor/comunication"
)

type Notification struct {
	Device  comunication.Device
	Message string
}

type Notifier struct {
	notifications  chan Notification
	c              chan struct{}
	webhookPattern string
}

func NewNotifier(webhookPattern string) *Notifier {
	return &Notifier{notifications: make(chan Notification, 100), c: make(chan struct{}), webhookPattern: webhookPattern}
}

func (n *Notifier) InitBots(complexes []comunication.Complex) {
	for _, complexStruct := range complexes {
		bot, err := tgbotapi.NewBotAPI(complexStruct.BotToken)

		if err != nil {
			continue
		}

		webhookUrl := fmt.Sprintf(n.webhookPattern, complexStruct.BotIdentity)

		wh, err := tgbotapi.NewWebhook(webhookUrl)

		if err != nil {
			continue
		}

		_, err = bot.Request(wh)

		if err != nil {
			continue
		}
	}
}

func (n *Notifier) Notify(notification Notification) {
	if notification.Device.Complex.NotificationEnabled {
		n.notifications <- notification
	}
}

func (n *Notifier) Start() {
	for {
		select {
		case <-n.c:
			return
		case notification := <-n.notifications:
			n.Send(notification)
		}
	}
}

func (n *Notifier) Send(notification Notification) {
	if notification.Device.Complex.NotificationEnabled == false {
		return
	}

	if notification.Device.NotificationEnabled == false && notification.Device.HasDirectWire == true {
		return
	}

	token := notification.Device.Complex.BotToken
	bot, err := tgbotapi.NewBotAPI(token)

	if err != nil {
		return
	}

	for _, channel := range notification.Device.Complex.BotChannels {
		msg := tgbotapi.NewMessage(channel, notification.Message)
		fmt.Printf("Sending message '%s' to %d\n", notification.Message, channel)
		_, _ = bot.Send(msg)
	}
}

func (n *Notifier) Stop() {
	n.c <- struct{}{}
}
