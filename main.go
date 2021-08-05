package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"

	_ "github.com/go-sql-driver/mysql"
)

// 디스코드로 들어오는 메시지의 시간대는 UTC+0.
// 로컬 시간대로 맞춰서 DB에 넣도록 시간대를 잘 바꾸어 잘 포맷된 형태로 반환한다.
func getTime(discordtime discordgo.Timestamp) string {
	temp, _ := discordtime.Parse()
	nowZoneName, nowZoneOff := time.Now().Zone()
	temp = temp.In(time.FixedZone(nowZoneName, nowZoneOff))
	return temp.Format("2006-01-02 15:04:05.999999")
}

// 메인 함수가 종료될 때 실행되는 함수. 각 채널별로 봇 종료 메시지를 날린다
func maindead(connInfo connInfoType, discord *discordgo.Session) {
	db := connectMySQL(connInfo)
	defer db.Close()

	result, e := db.Query(`SELECT channel_id FROM ` + san(connInfo.table) + ` GROUP BY channel_id`)
	if e != nil {
		return
	}

	channelIDs := []string{}

	for result.Next() {
		temp := ""
		if e := result.Scan(&temp); e != nil {
			return
		}
		channelIDs = append(channelIDs, temp)
	}

	for _, id := range channelIDs {
		discord.ChannelMessageSend(id, "봇 종료!")
	}
}

func main() {
	// 비밀스러운 정보들 불러오기
	tokenTemp, e := ioutil.ReadFile("token.txt")
	if e != nil {
		panic(e)
	}
	token := string(tokenTemp)

	pwTemp, e := ioutil.ReadFile("password.txt")
	if e != nil {
		panic(e)
	}
	password := string(pwTemp)

	// create discord session
	discord, e := discordgo.New("Bot " + token)
	if e != nil {
		log.Fatalln("error creating Discord session,", e)
		return
	}

	// MySQL 접속 정보
	connInfo := connInfoType{
		id:      "root",
		pw:      password,
		address: "127.0.0.1",
		port:    "3306",
		db:      "eraser_bot",
		table:   "bot_table",
	}

	initDB(connInfo)
	log.Println("initDB 함수 완료")

	// 주기적으로 메시지를 지우기 위해 cronjob을 넣는다
	c := cron.New(cron.WithSeconds())
	c.Start()
	// 5초마다 cronjob으로...
	c.AddFunc("*/5 * * * * *", func() {
		cronDelete(connInfo, discord, 60) // 60분 넘어가는 메시지를 지우기를 시도한다
	})

	// 메인 함수가 종료되면 실행될 것들
	defer func() {
		maindead(connInfo, discord)
		c.Stop()
		discord.Close()
	}()

	// init 과정을 실행할 수 있는 칸을 위한 틀 제작
	type didStartType struct {
		lock *sync.Mutex
		did  bool
	}
	didStart := make(map[string]didStartType)

	discord.AddHandler(func(session *discordgo.Session, message *discordgo.MessageCreate) {
		go func(sess *discordgo.Session, msg *discordgo.MessageCreate) {
			// 각 채널별로 init 과정을 실행할 수 있는 칸이 만들어졌는지 확인한다
			if _, ok := didStart[message.ChannelID]; !ok {
				didStart[message.ChannelID] = didStartType{
					lock: &sync.Mutex{},
					did:  false,
				}
			}

			// init 과정을 각 채널별로 실행했는지 기록한다
			didStart[message.ChannelID].lock.Lock()
			if !didStart[message.ChannelID].did {
				treatInitialStart(connInfo, sess, msg)
				didStart[message.ChannelID] = didStartType{
					lock: didStart[message.ChannelID].lock,
					did:  true,
				}
				didStart[message.ChannelID].lock.Unlock()
				// init 과정을 했다면 핸들러 함수를 종료한다
				return
			}
			didStart[message.ChannelID].lock.Unlock()

			// 들어오는 메시지 타입에 따라 실행 함수를 나눈다
			switch {
			case msg.Content == "?" && session.State.User.ID != message.Author.ID:
				result := "```"
				result += ".   방금 쳤던 내 메시지 지우는 명령\n"
				result += "..  자기가 쳤던거 전부 지우는 명령\n"
				result += "... 그냥 전부 지우는 명령\n"
				result += "```"

				sess.ChannelMessageSend(msg.ChannelID, result)

			case msg.Content == "." && session.State.User.ID != message.Author.ID:
				if session.State.User.ID == message.Author.ID {
					return
				}
				go treatDeleteSingle(connInfo, sess, msg)

			case msg.Content == ".." && session.State.User.ID != message.Author.ID:
				if session.State.User.ID == message.Author.ID {
					return
				}
				go treatDeleteMe(connInfo, sess, msg)

			case msg.Content == "..." && session.State.User.ID != message.Author.ID:
				if session.State.User.ID == message.Author.ID {
					return
				}
				go treatDeleteAll(connInfo, sess, msg)

			default:
				go treatNormalMessage(connInfo, sess, msg)
			}

		}(session, message)
	})

	// 세팅을 다 했으니 세션을 연다
	if e = discord.Open(); e != nil {
		log.Fatalln("error opening connection,", e)
		return
	}

	// Ctrl+C를 받아서 프로그램 자체를 종료하는 부분. os 신호를 받는다
	log.Println("bot is now running. Press Ctrl+C to exit.")
	{
		sc := make(chan os.Signal, 1)
		signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
		<-sc
	}
	log.Println("received Ctrl+C, please wait.")

}
