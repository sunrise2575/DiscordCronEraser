package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

func treatInitialStart(sess *discordgo.Session, channel *discordgo.Channel) {
	db := dbConnect()
	defer db.Close()

	dbTx(db, func(tx *sql.Tx) bool {
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
			query := "INSERT OR IGNORE INTO bot_table (channel_id, author_id, timestamp, message_id) VALUES "
			for i, m := range messages {
				query += fmt.Sprintf("(%v,%v,'%v',%v)", m.ChannelID, m.Author.ID, getTime(m.Timestamp), m.ID)
				if i < len(messages)-1 {
					query += ","
				}
			}

			dbTxExec(tx, query)

			// 넣은 메시지 정보 중 가장 오래된 메시지를 찾는다
			result := dbTxQuery(tx, `
					SELECT message_id
					FROM bot_table
					WHERE channel_id = ?
					ORDER BY timestamp ASC LIMIT 1
					`, channel.ID)

			if len(result) > 0 {
				beforeID = result[0][0]
			}
		}

		return true
	})
}
