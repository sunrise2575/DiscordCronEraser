package main

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

func treatInitialStart(connInfo connInfoType, sess *discordgo.Session, channel *discordgo.Channel) {
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

		messages, _ := sess.ChannelMessages(channel.ID, 100, beforeID, "", "")

		// 읽을 메시지가 없으면 반복문을 나온다
		if len(messages) == 0 {
			break
		}

		guild, _ := sess.Guild(channel.GuildID)
		log.Printf("read message: %v, [%v] in [%v]\n", len(messages), channel.Name, guild.Name)

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
					`, channel.ID)
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

	// 트랜잭션 종료
	if e := tx.Commit(); e != nil {
		panic(e)
	}
}
