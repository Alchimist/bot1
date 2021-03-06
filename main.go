package main

import (
	"encoding/json"
	"flag"
	"fmt"
//	"github.com/Syfaro/telegram-bot-api"
	"io/ioutil"
	"log"
	"net/http"
	"time"
	"os"
	"crypto/tls"
	"gopkg.in/telegram-bot-api.v4"
//	"strconv"
//	"strings"
//	"encoding/xml"
)

//
//Site list map  "URL" - status
//Site status
//0 - never checked
//1 - timeout
//200 - ok
//other statuses - crit

var (
	SiteList   map[string]int
	botToken   map[string]interface{}
	chatID     int64
	telegramBotToken string
	configFile string
	configFileBot string
	HelpMsg    = "Это простой мониторинг доступности сайтов. Он обходит сайты в списке и ждет что он ответит 200, если возвращается не 200 или ошибки подключения, то бот пришлет уведомления в групповой чат\n" +
		"Список доступных комманд:\n" +
		"/site_list - покажет список сайтов в мониторинге и их статусы (про статусы ниже)\n" +
		"/site_add [url] - добавит url в список мониторинга\n" +
		"/site_del [url] - удалит url из списка мониторинга\n" +
		"/help - отобразить это сообщение\n" +
		"\n" +
		"У сайтов может быть несколько статусов:\n" +
		"0 - никогда не проверялся (ждем проверки)\n" +
		"1 - ошибка подключения \n" +
		"200 - ОК-статус" +
		"все остальные http-коды считаются некорректными"
)

func init() {
	SiteList = make(map[string]int)

//	file, _ := os.Open("config_bot.json")
//	decoder := json.NewDecoder(file)
//	configuration := Config_bot{}
//	err := decoder.Decode(&configuration)
//	if err != nil {
//		log.Panic(err)
//	}
//	fmt.Println(configuration.TelegramBotToken)

	flag.StringVar(&configFileBot, "config_bot", "config_bot.json", "config file bot")
	flag.StringVar(&configFile, "config", "config.json", "config file")
//	flag.StringVar(&telegramBotToken, "telegrambottoken", "", "Telegram Bot Token")
//	flag.Int64Var(&chatID, "chatid", 0, "chatId to send messages")

	flag.Parse()

	load_list()

	telegramBotToken = botToken["TelegramBotToken"].(string) // "400069657:AAHldU0VZ7ZSfTSU55jnYtJpVnSdvgAqiyM"//
	if telegramBotToken == "" {
		log.Print("-telegrambottoken is required")
		os.Exit(1)
	}

//	chatID = -263587509
	chatID = int64(botToken["chatID"].(float64))
	if chatID == 0 {
		log.Print("-chatid is required")
		os.Exit(1)
	}

}

func send_notifications(bot *tgbotapi.BotAPI) {
	for site, status := range SiteList {
		if status != 200 {
			alarm := fmt.Sprintf("CRIT - %s ; status: %v", site, status)
			bot.Send(tgbotapi.NewMessage(chatID, alarm))
		}
	}
}

func save_list() {
	data, err := json.Marshal(SiteList)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(configFile, data, 0644)
	if err != nil {
		panic(err)
	}
}

func load_list() {
	data, err := ioutil.ReadFile(configFile)
	databot, err1 := ioutil.ReadFile(configFileBot)

	if err != nil {
		log.Printf("No such file - starting without config")
		return
	}

	if err1 != nil {
		log.Printf("No such file - starting without config bot")
		return
	}

	if err = json.Unmarshal(data, &SiteList); err != nil {
		log.Printf("Cant read file - starting without config")
		return
	}

	if err = json.Unmarshal(databot, &botToken); err != nil {
		log.Printf("Cant read file - starting without configbot")
		return
	}

//	fmt.Println(databot)

	fmt.Printf("тип: %T\n", botToken["TelegramBotToken"])
	fmt.Printf("тип: %T\n", int64(botToken["chatID"].(float64)))

	log.Printf(string(data))
}

func monitor(bot *tgbotapi.BotAPI) {

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	var httpclient = &http.Client{
		Timeout: time.Second * 10,
		Transport: tr,
	}

	for {
		save_list()
		for site, _ := range SiteList {
			response, err := httpclient.Get(site)
			if err != nil {
				SiteList[site] = 1
				log.Printf("Status of %s: %s", site, "1 - Connection refused")
			} else {
				log.Printf("Status of %s: %s", site, response.Status)
				SiteList[site] = response.StatusCode
			}
		}
		send_notifications(bot)
		time.Sleep(time.Minute * 5)
	}
}

func main() {
	bot, err := tgbotapi.NewBotAPI(telegramBotToken)
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)
	log.Printf("Config file: %s", configFile)
	log.Printf("Config file: %s", configFileBot)
	log.Printf("ChatID: %v", chatID)
	log.Printf("Starting monitoring thread")
	go monitor(bot)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprint("Я живой; вот сайты которые буду мониторить: ", SiteList)))

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		reply := "Не знаю что сказать"
		if update.Message == nil {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		switch update.Message.Command() {
		case "site_list":
			sl, _ := json.Marshal(SiteList)
			reply = string(sl)

		case "site_add":
			SiteList[update.Message.CommandArguments()] = 0
			reply = "Site added to monitoring list"

		case "site_del":
			delete(SiteList, update.Message.CommandArguments())
			reply = "Site deleted from monitoring list"
		case "help":
			reply = HelpMsg
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, reply)
		bot.Send(msg)
	}
}
