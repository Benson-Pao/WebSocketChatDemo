package main

import (
	fileIo "IOCopyByWS/io"
	"IOCopyByWS/model"
	"IOCopyByWS/server"
	"IOCopyByWS/service"
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"
)

var (
	//https://zh.stripchat.com/api/front/v2/config/data?requestPath=%2Fz_xinxin&timezoneOffset=-480&timezone=Asia%2FTaipei&defaultTag=girls&uniq=kqr9p34diyb5h8t6
	resource   string
	AuthUrl    string
	WSPort     string
	Csr        string
	key        string
	isRun      string
	configFile = "./config.json"

	config *model.ConfigInfo
)

func init() {
	log.SetFlags(log.Lshortfile | log.Ltime | log.Ldate)

	flag.StringVar(&resource, "resource", "https://zh.stripchat.com/api/front/models?limit=60&offset=0&primaryTag=girls&filterGroupTags=[[%22tagLanguageChinese%22]]&sortBy=stripRanking&parentTag=tagLanguageChinese&userRole=guest", "Resource URL")
	flag.StringVar(&AuthUrl, "auth", "https://zh.stripchat.com/api/front/v2/config/data", "Auth URL")
	flag.StringVar(&WSPort, "p", "443", "WSPort")
	flag.StringVar(&Csr, "csr", "./certificate/server.csr", "Certificate csr file")
	flag.StringVar(&key, "key", "./certificate/server.key", "Certificate key file")
	flag.StringVar(&isRun, "isRun", "1", "Service Run 1 Run 0 Stop")
	flag.Parse()
}

func main() {

	defer func() {
		if r := recover(); r != nil {
			log.Println("server panic: ", r)
		}
	}()

	config = &model.ConfigInfo{}

	if ok, err := fileIo.Exists(configFile); err != nil || !ok {
		config.ResourceURL = resource
		config.ResourceAuth = AuthUrl
		config.WSPort = WSPort
		config.Csr = Csr
		config.Key = key
		config.IsRun = isRun

		cfgJson, _ := json.Marshal(config)
		fileIo.Save(configFile, cfgJson)
	} else {
		file, _ := fileIo.Read(configFile)
		json.Unmarshal(file, config)
	}

	runtime.GOMAXPROCS(runtime.NumCPU())
	cxt, cancel := context.WithCancel(context.Background())

	closeCh := make(chan bool, 1)

	serv := service.NewService(cxt, config)
	chat := service.NewChat(serv, closeCh)
	_, err := server.NewServer(serv, chat)
	if err != nil {
		log.Println(err)
		return
	}

	//if err := db.LoadConnectFile(); err != nil {
	//	return
	//}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		for ch := range signalCh {
			switch ch {
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				log.Println("Program Exit...", ch)
				close(closeCh)
				cancel()
				time.Sleep(1 * time.Second)

				os.Exit(0)
			default:
				log.Println("signal.Notify default")

			}
		}
	}()

	log.Println("please input")
	for {
		log.Print(">")
		reader := bufio.NewReader(os.Stdin)
		text, _, _ := reader.ReadLine()

		switch strings.ToLower(string(text)) {
		case "run":
			config.IsRun = "1"
		case "stop":
			config.IsRun = "0"
		case "pause":
			serv.IsPause = true
			log.Println("Timer Pause")
		case "count":
			log.Println("Chats:", chat.Chat.Count)
			log.Println("Users:", chat.User.Count)
		case "chats":
			chat.Chat.List.Range(func(key, value any) bool {
				c := value.(*service.ChatInfo)

				log.Println("Key:", key, " Users:", c.Count)
				return true
			})
		case "users":
			chat.User.List.Range(func(key, value any) bool {
				u := value.(*service.UserInfo)
				log.Println("UID:", u.UID, " CID:", u.CID, " Status:", u.Status)
				return true
			})
		case "start":
			serv.IsPause = false
			log.Println("Timer Start")
		case "quit":
			close(closeCh)
			cancel()
			//s.Close()
			time.Sleep(1 * time.Second)

			return

		}
	}
}
