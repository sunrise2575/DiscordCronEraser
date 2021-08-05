package main

import (
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

func cronDelete(connInfo connInfoType, sess *discordgo.Session, minute int) {
	db := connectMySQL(connInfo)
	defer db.Close()

	// 트랜잭션 시작
	tx, e := db.Begin()
	if e != nil {
		panic(e)
	}

	chanMsgIDs := make(map[string][]string)
	now := time.Now().Format("2006-01-02 15:04:05.999999")

	exist := false
	func() {
		result, e := tx.Query(`
			SELECT channel_id, message_id
			FROM `+san(connInfo.table)+`
			WHERE timestamp < DATE_SUB(?, INTERVAL ? MINUTE)
			`, now, minute)
		if e != nil {
			panic(e)
		}
		defer result.Close()

		for result.Next() {
			exist = true
			channelID, messageID := "", ""
			if e := result.Scan(&channelID, &messageID); e != nil {
				panic(e)
			}
			if _, ok := chanMsgIDs[channelID]; !ok {
				chanMsgIDs[channelID] = []string{}
			}
			chanMsgIDs[channelID] = append(chanMsgIDs[channelID], messageID)
		}

		if !exist {
			if e := tx.Rollback(); e != nil {
				panic(e)
			}
		}
	}()

	if !exist {
		// already rollbacked
		return
	}

	{
		for channelID, messageIDs := range chanMsgIDs {
			// bulk delete 실행
			if e := sess.ChannelMessagesBulkDelete(channelID, messageIDs); e != nil {
				if e := tx.Rollback(); e != nil {
					panic(e)
				}
				return
			}
		}
	}

	{
		result, e := tx.Exec(`
			DELETE FROM `+san(connInfo.table)+`
			WHERE timestamp < DATE_SUB(?, INTERVAL ? MINUTE)
			`, now, minute)
		if e != nil {
			panic(e)
		}

		affected, e := result.RowsAffected()
		if e != nil {
			panic(e)
		}
		if affected > 0 {
			log.Printf("%v개의 오래된 메시지가 삭제됨!", affected)
		}
	}

	// 트랜잭션 종료
	if e := tx.Commit(); e != nil {
		panic(e)
	}
}
