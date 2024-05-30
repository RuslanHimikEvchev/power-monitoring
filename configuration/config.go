package configuration

import (
	"github.com/gin-gonic/gin"
	"go-meshtastic-monitor/comunication"
	"go-meshtastic-monitor/core"
)

type Configuration struct {
	Redis                  core.RedisConf         `yaml:"redis"`
	TelegramWebhookPattern string                 `yaml:"telegram_webhook_pattern"`
	Complexes              []comunication.Complex `yaml:"complexes"`
	ConfigRereadInterval   int64                  `yaml:"config_reread_interval"`
	HttpBind               string                 `yaml:"http_bind"`
	HttpSecurity           gin.Accounts           `yaml:"http_security"`
}
