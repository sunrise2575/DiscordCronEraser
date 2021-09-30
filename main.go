package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/signal"
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

// 메인 함수가 종료될 때 실행되는 함수.
func shutdownProcedure(connInfo connInfoType) {
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
}

func readFileAsString(path string) string {
	out, e := ioutil.ReadFile(path)
	if e != nil {
		panic(e)
	}
	return string(out)
}

func main() {
	// MySQL 접속 정보
	connInfo := connInfoType{
		id:      "root",
		pw:      readFileAsString("mysql_password.txt"),
		address: "127.0.0.1",
		port:    "3306",
		db:      "eraser_bot",
		table:   "bot_table",
	}

	initDB(connInfo)
	log.Println("initialize DB complete")

	// create discord session
	discord, e := discordgo.New("Bot " + readFileAsString("./token.txt"))
	if e != nil {
		log.Fatalln("error creating Discord session,", e)
		return
	}

	// 주기적으로 메시지를 지우기 위해 cronjob을 넣는다
	c := cron.New(cron.WithSeconds())
	c.Start()
	// 5초마다 cronjob으로...
	c.AddFunc("*/5 * * * * *", func() {
		cronDelete(connInfo, discord, 60) // 60분 넘어가는 메시지를 지우기를 시도한다
	})

	commands := makeCommands(connInfo)

	// 세팅을 다 했으니 세션을 연다
	if e = discord.Open(); e != nil {
		log.Fatalln("error opening connection,", e)
		return
	}

	// 메인 함수가 종료되면 실행될 것들
	defer func() {
		shutdownProcedure(connInfo)
		c.Stop()
		discord.Close()
		log.Println("bye")
	}()

	// Guild에 초대 / 접속했을 때 실행하는 부분
	discord.AddHandler(func(session *discordgo.Session, event *discordgo.GuildCreate) {
		// Guild에 허용된 채널 읽기
		for _, channel := range event.Guild.Channels {
			if channel.Type == discordgo.ChannelTypeGuildText {
				permission, e := session.UserChannelPermissions(discord.State.User.ID, channel.ID)
				if e != nil {
					log.Println(e)
				}
				mustRequired := discordgo.PermissionViewChannel |
					discordgo.PermissionSendMessages |
					discordgo.PermissionManageMessages |
					discordgo.PermissionReadMessageHistory

				if permission&int64(mustRequired) == int64(mustRequired) {
					treatInitialStart(connInfo, discord, channel)
					log.Printf("initialize complete: [%v] in [%v]\n", channel.Name, event.Guild.Name)
				}
			}
		}
	})

	discord.AddHandler(func(sess *discordgo.Session, msg *discordgo.MessageCreate) {
		if cmd, ok := commands[msg.Content]; ok {
			if e := sess.ChannelMessageDelete(msg.ChannelID, msg.ID); e != nil {
				log.Println(e)
			}
			if msg.Author.ID != discord.State.User.ID {
				cmd.Callback(sess, msg.Message)
			}
		} else {
			treatMessageCreate(connInfo, sess, msg)
		}
	})

	// Ctrl+C를 받아서 프로그램 자체를 종료하는 부분. os 신호를 받는다
	log.Println("bot is now running. Press Ctrl+C to exit.")
	{
		sc := make(chan os.Signal, 1)
		signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
		<-sc
	}
	log.Println("received Ctrl+C, please wait.")
}
