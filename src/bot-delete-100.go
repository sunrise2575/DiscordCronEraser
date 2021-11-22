package main

import (
	"database/sql"
	"log"

	"github.com/bwmarrin/discordgo"
)

func treatDelete100(sess *discordgo.Session, msg *discordgo.Message) {
	db := dbConnect()
	defer db.Close()

	dbTx(db, func(tx *sql.Tx) bool {
		messageIDs := []string{}

		result := dbTxQuery(tx, `
			SELECT message_id
			FROM bot_table 
			WHERE channel_id = ? 
			LIMIT 100
			`, msg.ChannelID)

		for _, v := range result {
			messageIDs = append(messageIDs, v[0])
		}

		if len(messageIDs) == 0 {
			return false // rollback
		}

		if e := sess.ChannelMessagesBulkDelete(msg.ChannelID, messageIDs); e != nil {
			log.Println(e)
			return false // rollback
		}

		// 가장 최근의 메시지를 지운다
		affected := dbTxExec(tx, `
			DELETE FROM bot_table
			WHERE message_id IN (
				SELECT message_id
				FROM bot_table 
				WHERE channel_id = ? 
				LIMIT 100
			)
			`, msg.ChannelID)

		if affected > 0 {
			log.Printf("delete %v message(s) at treatDelete100 by user '%v' (user ID: %v, channel ID: %v)",
				affected, msg.Author.Username, msg.Author.ID, msg.ChannelID)
		}

		return true
	})
}
