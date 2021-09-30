package main

import (
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func treatDeleteSingle(connInfo connInfoType, sess *discordgo.Session, msg *discordgo.Message) {
	db := connectMySQL(connInfo)
	defer db.Close()

	// 트랜잭션 시작
	tx, e := db.Begin()
	if e != nil {
		panic(e)
	}

	exist := false
	var messageID int
	func() {
		result, e := tx.Query(`
			SELECT message_id
			FROM `+san(connInfo.table)+`
			WHERE channel_id = ? AND author_id = ?
			ORDER BY timestamp DESC LIMIT 1
			`, msg.ChannelID, msg.Author.ID)
		if e != nil {
			panic(e)
		}
		defer result.Close()

		for result.Next() {
			exist = true
			if e := result.Scan(&messageID); e != nil {
				panic(e)
			}
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
		// 지운다
		if e := sess.ChannelMessageDelete(msg.ChannelID, strconv.Itoa(messageID)); e != nil {
			// 이미 지워졌는지 아닌지 판단
			if !strings.Contains(e.Error(), `"code": 10008`) {
				// 이미 지워졌는데 오류가 났다? 뭔가 이상하다
				if e := tx.Rollback(); e != nil {
					panic(e)
				}
				return
			}
		}
	}

	{
		// 기억에서 가장 최근의 메시지를 지운다
		_, e := tx.Exec(`
			DELETE FROM `+san(connInfo.table)+`
			WHERE channel_id = ? AND author_id = ? AND message_id = ?
			`, msg.ChannelID, msg.Author.ID, messageID)
		if e != nil {
			panic(e)
		}
	}

	// 트랜잭션 종료
	if e := tx.Commit(); e != nil {
		panic(e)
	}
}
