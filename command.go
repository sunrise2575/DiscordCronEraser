package main

import (
	"github.com/bwmarrin/discordgo"
)

type DiscordCommand struct {
	Description string
	Callback    func(sess *discordgo.Session, msg *discordgo.Message)
}

type DiscordCommands map[string]DiscordCommand

func makeCommands(connInfo connInfoType) DiscordCommands {
	// 커맨드 만든다
	result := make(DiscordCommands)

	result["."] = DiscordCommand{
		Description: "직전에 입력한 내 메시지 1개를 삭제합니다",
		Callback: func(sess *discordgo.Session, msg *discordgo.Message) {
			treatDeleteSingle(connInfo, sess, msg)
		},
	}

	result[".."] = DiscordCommand{
		Description: "내 메시지만 모두 삭제합니다",
		Callback: func(sess *discordgo.Session, msg *discordgo.Message) {
			treatDeleteMe(connInfo, sess, msg)
		},
	}

	result["..."] = DiscordCommand{
		Description: "내 메시지만 모두 삭제합니다",
		Callback: func(sess *discordgo.Session, msg *discordgo.Message) {
			if msg.Author.ID != "293241938444943362" {
				return
			}
			treatDeleteAll(connInfo, sess, msg)
		},
	}

	result["??"] = DiscordCommand{
		Description: "물어본다",
		Callback: func(sess *discordgo.Session, msg *discordgo.Message) {
			content := "```"
			for name, cmd := range result {
				content += name
				content += "\t"
				content += cmd.Description
				content += "\n"
			}
			content += "```"
			sess.ChannelMessageSend(msg.ChannelID, content)
		},
	}

	return result
}
