package main

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ilyakaznacheev/cleanenv"
	"go-meshtastic-monitor/comunication"
	"go-meshtastic-monitor/configuration"
	"go-meshtastic-monitor/core"
	"go-meshtastic-monitor/direct_wire"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var (
	config       configuration.Configuration
	confFilePath string
)

func init() {
	var err error

	if len(os.Args) < 2 || (len(os.Args) > 1 && os.Args[1] == "") {
		log.Fatal("[MAIN] no path to config defined")
	}

	confFilePath = strings.TrimSpace(os.Args[1])

	err = cleanenv.ReadConfig(confFilePath, &config)

	if err != nil {
		panic(err)
	}
}

func parseComplexes() []comunication.Complex {
	var conf configuration.Configuration
	err := cleanenv.ReadConfig(confFilePath, &conf)

	if err != nil {
		panic(err)
	}

	return conf.Complexes
}

func findComplex(complexes []comunication.Complex, bot string) comunication.Complex {
	for _, c := range complexes {
		if c.BotIdentity == bot {
			return c
		}
	}

	return comunication.Complex{}
}

func main() {
	redisConnect := core.NewRedisConnect(config.Redis)
	storage := core.NewRedisStorage(redisConnect)

	n := core.NewNotifier(config.TelegramWebhookPattern)
	monitor := core.NewMonitor(config.Complexes, n, storage)
	onlineMonitor := direct_wire.NewDirectWireMonitor(n, config.Complexes, storage)
	n.InitBots(config.Complexes)

	monitor.Restore()
	onlineMonitor.Restore()

	ticker := time.NewTicker(time.Duration(config.ConfigRereadInterval) * time.Second)
	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				monitor.UpdateComplexes(parseComplexes())
				n.InitBots(parseComplexes())
				onlineMonitor.UpdateComplexes(parseComplexes())
			case <-stop:
				return
			}
		}
	}()

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	keepAlive := make(chan os.Signal)
	signal.Notify(keepAlive, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGQUIT, syscall.SIGHUP, syscall.SIGUSR1, syscall.SIGUSR2)

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{})
	})

	auth := r.Group("/admin", gin.BasicAuth(config.HttpSecurity))
	auth.GET("/status", func(c *gin.Context) {
		c.JSON(200, monitor.GetStatus())
	})
	auth.GET("/online-status", func(c *gin.Context) {
		c.JSON(200, onlineMonitor.GetStatus())
	})
	auth.GET("/update-notification", func(c *gin.Context) {
		mac := c.Query("mac")
		status := c.Query("status")

		if mac != "" && status != "" {
			notificationEnabled := status == direct_wire.StatusOn

			onlineMonitor.UpdateNotification(mac, notificationEnabled)
		}

		c.JSON(200, onlineMonitor.GetStatus())
	})

	r.Any("/:bot/webhook", func(context *gin.Context) {
		c := findComplex(parseComplexes(), context.Param("bot"))

		if c.Key == "" {
			context.JSON(http.StatusOK, gin.H{"error": "bot not found"})

			return
		}

		bot, err := tgbotapi.NewBotAPI(c.BotToken)

		if err != nil {
			context.JSON(http.StatusOK, gin.H{"error": "bot token err " + err.Error()})

			return
		}

		u, err := bot.HandleUpdate(context.Request)

		if err != nil {
			context.JSON(http.StatusOK, gin.H{"error": "bot update err " + err.Error()})

			return
		}

		if u.Message == nil {
			context.JSON(http.StatusOK, gin.H{"error": "bot message nil"})

			return
		}

		if u.Message.IsCommand() {
			msg := tgbotapi.NewMessage(u.Message.Chat.ID, "")

			switch u.Message.Command() {
			case "start":
				msg.Text = "Вітаю!"
				break
			case "schedule":

			case "status":
				var text string
				if c.IsDirectWire {
					text = onlineMonitor.GetStatusText(c)
				} else {
					text = monitor.GetStatusText(c)
				}

				if text == "" {
					text = "Нічого не знайдено"
				}

				msg.Text = text
				break
			default:
				msg.Text = "Невідома команда. Спробуйте /status"
			}

			_, _ = bot.Send(msg)
		}

		context.JSON(http.StatusOK, gin.H{"error": "bot message nil"})
	})

	r.POST("/direct-wire", func(c *gin.Context) {
		d, err := parseDevice(c)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		onlineMonitor.HandleDevice(d)

		c.JSON(200, gin.H{"message": "ok"})
	})

	r.POST("/timeout-api", func(context *gin.Context) {
		d, err := parseDevice(context)

		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		monitor.AddDevice(d)

		context.JSON(200, gin.H{"message": "ok"})
	})

	go r.Run(config.HttpBind)
	go n.Start()

	go monitor.Start()

	<-keepAlive
	monitor.Stop()
	n.Stop()
	monitor.Backup()
	onlineMonitor.Backup()
	stop <- struct{}{}
}

func parseDevice(c *gin.Context) (comunication.Device, error) {
	d := comunication.Device{}
	b, err := c.GetRawData()

	if err != nil {
		return d, err
	}
	err = json.Unmarshal(b, &d)
	if err != nil {
		return d, err
	}

	return d, nil
}
