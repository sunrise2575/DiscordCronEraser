package main

import (
	"database/sql"
	"log"

	"github.com/bwmarrin/discordgo"
)

func treatDeleteMe(sess *discordgo.Session, msg *discordgo.Message) {
	db := dbConnect()
	defer db.Close()

	complete := false
	for !complete {
		dbTx(db, func(tx *sql.Tx) bool {
			messageIDs := []string{}

			result := dbTxQuery(tx, `
				SELECT message_id
				FROM bot_table
				WHERE channel_id = ? AND author_id = ?
				LIMIT 100
				`, msg.ChannelID, msg.Author.ID)

			for _, row := range result {
				messageIDs = append(messageIDs, row[0])
			}

			if len(messageIDs) == 0 {
				complete = true
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
					WHERE channel_id = ? AND author_id = ?
					LIMIT 100
				)
				`, msg.ChannelID, msg.Author.ID)

			if affected > 0 {
				log.Printf("delete %v message(s) at treatDeleteMe by user '%v' (user ID: %v, channel ID: %v)",
					affected, msg.Author.Username, msg.Author.ID, msg.ChannelID)
			}

			return true // commit
		})
	}
}
