package main

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

func treatDeleteMe(connInfo connInfoType, sess *discordgo.Session, msg *discordgo.MessageCreate) {
	// 명령어 그 자체는 즉시 삭제한다
	if e := sess.ChannelMessageDelete(msg.ChannelID, msg.ID); e != nil {
		log.Println(e)
	}

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
			WHERE channel_id = ? AND author_id = ?
			`, msg.ChannelID, msg.Author.ID)
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
		// bulk delete 실행
		if e := sess.ChannelMessagesBulkDelete(msg.ChannelID, messageIDs); e != nil {
			if e := tx.Rollback(); e != nil {
				panic(e)
			}
			return
		}
	}

	{
		// 기억에서 가장 최근의 메시지를 지운다
		_, e := tx.Exec(`
			DELETE FROM `+san(connInfo.table)+`
			WHERE channel_id = ? AND author_id = ?
			`, msg.ChannelID, msg.Author.ID)
		if e != nil {
			panic(e)
		}
	}

	// 트랜잭션 종료
	if e := tx.Commit(); e != nil {
		panic(e)
	}
}
