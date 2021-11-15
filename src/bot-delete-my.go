package main

import (
	"database/sql"

	"github.com/bwmarrin/discordgo"
)

func treatDeleteMe(sess *discordgo.Session, msg *discordgo.Message) {
	db := dbConnect()
	defer db.Close()

	dbTx(db, func(tx *sql.Tx) bool {
		exist := false
		messageIDs := []string{}
		result := dbTxQuery(tx, `
			SELECT message_id
			FROM bot_table
			WHERE channel_id = ? AND author_id = ?
			`, msg.ChannelID, msg.Author.ID)

		for _, row := range result {
			exist = true
			messageIDs = append(messageIDs, row[0])
		}

		if !exist {
			return false // rollback
		}

		// bulk delete 실행. 한번에 100개까지만 지우기 때문에 반복해서 지워야 한다
		for i := 0; i < len(messageIDs); i += 100 {
			if len(messageIDs)-i >= 100 {
				if e := sess.ChannelMessagesBulkDelete(msg.ChannelID, messageIDs[i:i+100]); e != nil {
					return false // rollback
				}
			} else {
				// 끝에 100개보다 덜 남은 경우
				if e := sess.ChannelMessagesBulkDelete(msg.ChannelID, messageIDs[i:]); e != nil {
					return false // rollback
				}
			}
		}

		// 가장 최근의 메시지를 지운다
		dbTxExec(tx, `
			DELETE FROM bot_table
			WHERE channel_id = ? AND author_id = ?
			`, msg.ChannelID, msg.Author.ID)

		return true // commit
	})
}