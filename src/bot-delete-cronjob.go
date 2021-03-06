package main

import (
	"database/sql"
	"log"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
)

func cronDelete(sess *discordgo.Session, minute int) {
	db := dbConnect()
	defer db.Close()

	// 트랜잭션 시작
	dbTx(db, func(tx *sql.Tx) bool {
		chanMsgIDs := make(map[string][]string)
		now := time.Now().Format("2006-01-02 15:04:05.999")

		exist := false

		result := dbTxQuery(tx, `
			SELECT channel_id, message_id
			FROM bot_table
			WHERE timestamp < datetime(?, '-`+strconv.Itoa(minute)+` minutes')
			LIMIT 100
		`, now)

		for _, row := range result {
			exist = true
			channelID, messageID := row[0], row[1]
			if _, ok := chanMsgIDs[channelID]; !ok {
				chanMsgIDs[channelID] = []string{}
			}
			chanMsgIDs[channelID] = append(chanMsgIDs[channelID], messageID)
		}

		if !exist {
			return false // rollback
		}

		for channelID, messageIDs := range chanMsgIDs {
			// bulk delete 실행
			if e := sess.ChannelMessagesBulkDelete(channelID, messageIDs); e != nil {
				log.Println(e)
				return false // rollback
			}
		}

		affected := dbTxExec(tx, `
			DELETE FROM bot_table
			WHERE message_id IN (
				SELECT message_id
				FROM bot_table
				WHERE timestamp < datetime(?, '-`+strconv.Itoa(minute)+` minutes')
				LIMIT 100
			)
			`, now)

		if affected > 0 {
			log.Printf("delete %v message(s) at cronDelete", affected)
		}

		return true // commit
	})
}
