package main

import (
	"github.com/bwmarrin/discordgo"
)

func treatMessageCreate(sess *discordgo.Session, msg *discordgo.MessageCreate) {
	db := dbConnect()
	defer db.Close()

	dbExec(db, `
		INSERT OR IGNORE INTO bot_table (channel_id,author_id,timestamp,message_id)
		VALUES (?,?,?,?)
		`, msg.ChannelID, msg.Author.ID, getTime(msg.Timestamp), msg.ID)
}
