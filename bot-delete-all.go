package main

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

func treatDeleteAll(connInfo connInfoType, sess *discordgo.Session, msg *discordgo.Message) {
	db := connectMySQL(connInfo)
	defer db.Close()

	// 트랜잭션 시작
	tx, e := db.Begin()
	if e != nil {
		panic(e)
	}

	exist := false
	messageIDs := []string{}
	func() {
		result, e := tx.Query(`
			SELECT message_id
			FROM `+san(connInfo.table)+`
			WHERE channel_id = ? 
			`, msg.ChannelID)
		if e != nil {
			panic(e)
		}
		defer result.Close()

		for result.Next() {
			exist = true
			temp := ""
			if e := result.Scan(&temp); e != nil {
				panic(e)
			}
			messageIDs = append(messageIDs, temp)
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
		// bulk delete 실행. 한번에 100개까지만 지우기 때문에 반복해서 지워야 한다
		for i := 0; i < len(messageIDs); i += 100 {
			if len(messageIDs)-i >= 100 {
				if e := sess.ChannelMessagesBulkDelete(msg.ChannelID, messageIDs[i:i+100]); e != nil {
					if e := tx.Rollback(); e != nil {
						panic(e)
					}
					return
				}
			} else {
				// 끝에 100개보다 덜 남은 경우
				if e := sess.ChannelMessagesBulkDelete(msg.ChannelID, messageIDs[i:]); e != nil {
					if e := tx.Rollback(); e != nil {
						panic(e)
					}
					return
				}
			}
			time.Sleep(time.Second * 2)
		}
	}

	{
		// 기억에서 가장 최근의 메시지를 지운다
		_, e := tx.Exec(`
			DELETE FROM `+san(connInfo.table)+`
			WHERE channel_id = ?
			`, msg.ChannelID)
		if e != nil {
			panic(e)
		}
	}

	// 트랜잭션 종료
	if e := tx.Commit(); e != nil {
		panic(e)
	}
}
