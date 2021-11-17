package main

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

func treatMessageCreate(sess *discordgo.Session, msg *discordgo.MessageCreate) {
	db := dbConnect()
	defer db.Close()

	affected := dbExec(db, `
		INSERT OR IGNORE INTO bot_table (channel_id,author_id,timestamp,message_id)
		VALUES (?,?,?,?)
		`, msg.ChannelID, msg.Author.ID, getTime(msg.Timestamp), msg.ID)

	if affected > 0 {
		log.Printf("insert %v message(s) at treatMessageCreate by user '%v' (user ID: %v)", affected, msg.Author.Username, msg.Author.ID)
	}
}
