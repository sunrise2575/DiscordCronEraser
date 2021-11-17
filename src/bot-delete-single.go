package main

import (
	"database/sql"
	"log"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func treatDeleteSingle(sess *discordgo.Session, msg *discordgo.Message) {
	db := dbConnect()
	defer db.Close()

	dbTx(db, func(tx *sql.Tx) bool {
		result := dbTxQuery(tx, `
			SELECT message_id
			FROM bot_table
			WHERE channel_id = ? AND author_id = ?
			ORDER BY timestamp DESC LIMIT 1
			`, msg.ChannelID, msg.Author.ID)

		if len(result) == 0 {
			return false // rollback
		}

		messageID, e := strconv.Atoi(result[0][0])
		if e != nil {
			log.Println(e)
			return false // rollback
		}

		// 지운다
		if e := sess.ChannelMessageDelete(msg.ChannelID, strconv.Itoa(messageID)); e != nil {
			// 이미 지워졌는지 아닌지 판단
			if !strings.Contains(e.Error(), `"code": 10008`) {
				// 이미 지워졌는데 오류가 났다? 뭔가 이상하다
				log.Println(e)
				return false // rollback
			}
		}

		// 가장 최근의 메시지를 지운다
		affected := dbTxExec(tx, `
			DELETE FROM bot_table
			WHERE message_id = ?
			`, messageID)

		if affected > 0 {
			log.Printf("delete %v message(s) at treatDeleteSingle by user '%v' (user ID: %v, channel ID: %v)",
				affected, msg.Author.Username, msg.Author.ID, msg.ChannelID)
		}

		return true // commit
	})
}
