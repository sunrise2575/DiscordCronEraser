package main

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

func treatInitialStart(connInfo connInfoType, sess *discordgo.Session, msg *discordgo.MessageCreate) {
	// initial start for the server
	sess.ChannelMessageSend(msg.ChannelID, "봇 초기화 시작: 이전의 메시지를 읽고 있습니다")

	db := connectMySQL(connInfo)
	defer db.Close()

	// 트랜잭션 시작
	tx, e := db.Begin()
	if e != nil {
		panic(e)
	}

	beforeID := ""
	for {
		// 채널에 있는 메시지를 읽어온다

		messages, _ := sess.ChannelMessages(msg.ChannelID, 100, beforeID, "", "")

		// 읽을 메시지가 없으면 반복문을 나온다
		if len(messages) == 0 {
			break
		}

		log.Printf("초기화 중... 메시지 %v 개 읽음", len(messages))

		// 읽은 메시지 정보 중 필요한 정보만 bulk insert하는 query를 생성해서 실행한다
		query := "INSERT IGNORE INTO " + san(connInfo.table) + " (channel_id, author_id, timestamp, message_id) VALUES "
		for i, m := range messages {
			query += fmt.Sprintf("(%v,%v,'%v',%v)", m.ChannelID, m.Author.ID, getTime(m.Timestamp), m.ID)
			if i < len(messages)-1 {
				query += ","
			}
		}

		if _, e := tx.Exec(query); e != nil {
			panic(e)
		}

		// 넣은 메시지 정보 중 가장 오래된 메시지를 찾는다
		func() {
			result, e := tx.Query(`
					SELECT message_id
					FROM `+san(connInfo.table)+`
					WHERE channel_id = ?
					ORDER BY timestamp ASC LIMIT 1
					`, msg.ChannelID)
			if e != nil {
				panic(e)
			}
			defer result.Close()

			for result.Next() {
				if e := result.Scan(&beforeID); e != nil {
					panic(e)
				}
			}
		}()
	}

	// 완료라는 메시지를 출력하고 이것도 DB에 넣는다
	{
		m, e := sess.ChannelMessageSend(msg.ChannelID, "봇 초기화 완료")
		if e != nil {
			panic(e)
		}

		result, e := tx.Exec(`
			INSERT IGNORE INTO `+san(connInfo.table)+` (channel_id,author_id,timestamp,message_id)
			VALUES (?,?,?,?)
			`, m.ChannelID, m.Author.ID, getTime(m.Timestamp), m.ID)
		if e != nil {
			panic(e)
		}

		// 삽입이 잘못 이루어졌으면 프로그램 종료
		if _, e := result.RowsAffected(); e != nil {
			panic(e)
		}
	}

	// 트랜잭션 종료
	if e := tx.Commit(); e != nil {
		panic(e)
	}

	log.Println("초기화 성공")
}
