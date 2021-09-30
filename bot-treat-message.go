package main

import (
	"github.com/bwmarrin/discordgo"
)

func treatMessageCreate(connInfo connInfoType, sess *discordgo.Session, msg *discordgo.MessageCreate) {
	db := connectMySQL(connInfo)
	defer db.Close()

	{
		result, e := db.Exec(`
			INSERT IGNORE INTO `+san(connInfo.table)+` (channel_id,author_id,timestamp,message_id)
			VALUES (?,?,?,?)
			`, msg.ChannelID, msg.Author.ID, getTime(msg.Timestamp), msg.ID)
		if e != nil {
			panic(e)
		}

		if _, e := result.RowsAffected(); e != nil {
			panic(e)
		}
	}
}
