package main

import (
	"log"
	"sort"

	"github.com/bwmarrin/discordgo"
	"github.com/tidwall/gjson"
)

type DiscordCommand struct {
	Description string
	Callback    func(arg []string, sess *discordgo.Session, msg *discordgo.Message)
}

type DiscordCommands map[string]DiscordCommand

func makeCommands(conf gjson.Result) DiscordCommands {
	// 커맨드 만든다
	result := make(DiscordCommands)

	result["."] = DiscordCommand{
		Description: "직전에 입력한 내 메시지 1개를 삭제합니다",
		Callback: func(arg []string, sess *discordgo.Session, msg *discordgo.Message) {
			treatDeleteSingle(sess, msg)
		},
	}

	result[".."] = DiscordCommand{
		Description: "내 메시지만 최신 100개를 삭제합니다",
		Callback: func(arg []string, sess *discordgo.Session, msg *discordgo.Message) {
			treatDeleteMe100(sess, msg)
		},
	}

	result[".!"] = DiscordCommand{
		Description: "어떤 메시지든 최신 100개를 삭제합니다 (봇 만든 사람 전용)",
		Callback: func(arg []string, sess *discordgo.Session, msg *discordgo.Message) {
			if msg.Author.ID != conf.Get("admin_user_id").String() {
				log.Println("try to delete all message but failed:", msg.Author.Username)
				return
			}
			treatDelete100(sess, msg)
		},
	}

	result[".?"] = DiscordCommand{
		Description: "봇이 고장났을 경우 메시지를 다시 읽어 DB를 완전하게 만듭니다 (봇 만든 사람 전용)",
		Callback: func(arg []string, sess *discordgo.Session, msg *discordgo.Message) {
			if msg.Author.ID != conf.Get("admin_user_id").String() {
				log.Println("treat refresh but failed:", msg.Author.Username)
				return
			}
			treatRefresh(sess, msg)
		},
	}

	result["??"] = DiscordCommand{
		Description: "해법 출력",
		Callback: func(arg []string, sess *discordgo.Session, msg *discordgo.Message) {
			temp := [][2]string{}
			for name, cmd := range result {
				temp = append(temp, [2]string{name, cmd.Description})
			}
			sort.Slice(temp, func(i, j int) bool {
				return temp[i][0] < temp[j][0]
			})
			content := ">>> "
			for _, cmd := range temp {
				content += "`" + cmd[0] + "`"
				content += " : "
				content += cmd[1]
				content += "\n"
			}
			sess.ChannelMessageSend(msg.ChannelID, content)
		},
	}

	return result
}
