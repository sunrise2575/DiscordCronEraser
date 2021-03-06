package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
	"github.com/tidwall/gjson"

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

func readFileAsString(path string) string {
	out, e := ioutil.ReadFile(path)
	if e != nil {
		panic(e)
	}
	return string(out)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	conf := gjson.Parse(readFileAsString("../config.json"))

	db := dbConnect()
	if !dbTx(db, func(tx *sql.Tx) bool {
		dbTxExec(tx, `
		CREATE TABLE bot_table (
			channel_id BIGINT NOT NULL,
			author_id BIGINT NOT NULL,
			message_id BIGINT NOT NULL,
			timestamp DATETIME(6) NOT NULL,
			PRIMARY KEY (message_id)
		)`)
		dbTxExec(tx, `CREATE INDEX chan_idx ON bot_table (channel_id)`)
		dbTxExec(tx, `CREATE INDEX chan_auth_idx ON bot_table (channel_id, author_id)`)

		return true
	}) {
		log.Fatalln("initialize DB falied")
		return
	}
	log.Println("initialize DB complete")
	defer db.Close()

	// create discord session
	discord, e := discordgo.New("Bot " + conf.Get("bot_token").String())
	if e != nil {
		log.Fatalln("error creating Discord session,", e)
		return
	}

	// 주기적으로 메시지를 지우기 위해 cronjob을 넣는다
	c := cron.New(cron.WithSeconds())
	c.Start()
	// 5초마다 cronjob으로...
	c.AddFunc("*/5 * * * * *", func() {
		cronDelete(discord, int(conf.Get("minute").Int()))
	})

	commands := makeCommands(conf)

	// 세팅을 다 했으니 세션을 연다
	if e = discord.Open(); e != nil {
		log.Fatalln("error opening connection,", e)
		return
	}

	// 메인 함수가 종료되면 실행될 것들
	defer func() {
		c.Stop()
		discord.Close()
		log.Println("bye")
	}()

	if e := discord.UpdateStatusComplex(discordgo.UpdateStatusData{
		Activities: []*discordgo.Activity{
			{
				Name: fmt.Sprintf("%d분 넘는 메시지 삭제", int(conf.Get("minute").Int())),
				Type: discordgo.ActivityTypeGame,
			},
		},
	}); e != nil {
		log.Fatalln("error update status complex,", e)
		return
	}

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
					treatInitialStart(discord, channel)
					log.Printf("initialize complete: [%v] in [%v]\n", channel.Name, event.Guild.Name)
				}
			}
		}
	})

	discord.AddHandler(func(sess *discordgo.Session, msg *discordgo.MessageCreate) {
		if msg.Author.ID != discord.State.User.ID {
			// filter command message
			if len(msg.Content) > 0 {
				arg := strings.Fields(msg.Content)
				if len(arg) > 0 {
					if cmd, ok := commands[arg[0]]; ok {
						if e := sess.ChannelMessageDelete(msg.ChannelID, msg.ID); e != nil {
							log.Println(e)
						}
						cmd.Callback(arg, sess, msg.Message)
						return
					}
				}
			}
		}

		// normal message
		treatMessageCreate(sess, msg)
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
