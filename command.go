package main

import (
	"fmt"
	"log"
	"math/rand"
	"sort"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/seehuhn/mt19937"
)

type DiscordCommand struct {
	Description string
	Callback    func(arg []string, sess *discordgo.Session, msg *discordgo.Message)
}

type DiscordCommands map[string]DiscordCommand

func makeCommands(connInfo connInfoType) DiscordCommands {
	// 커맨드 만든다
	result := make(DiscordCommands)

	result["."] = DiscordCommand{
		Description: "직전에 입력한 내 메시지 1개를 삭제합니다",
		Callback: func(arg []string, sess *discordgo.Session, msg *discordgo.Message) {
			treatDeleteSingle(connInfo, sess, msg)
		},
	}

	result[".."] = DiscordCommand{
		Description: "내 메시지만 모두 삭제합니다",
		Callback: func(arg []string, sess *discordgo.Session, msg *discordgo.Message) {
			treatDeleteMe(connInfo, sess, msg)
		},
	}

	result["..."] = DiscordCommand{
		Description: "모든 메시지를 삭제합니다 (관리자 전용)",
		Callback: func(arg []string, sess *discordgo.Session, msg *discordgo.Message) {
			if msg.Author.ID != "293241938444943362" {
				log.Println("try to delete all message:", msg.Author.Username)
				return
			}
			treatDeleteAll(connInfo, sess, msg)
		},
	}

	result["rand"] = DiscordCommand{
		Description: "주사위를 굴린다. rand == [1,6], rand <n> == [1,n]",
		Callback: func(arg []string, sess *discordgo.Session, msg *discordgo.Message) {
			rng := rand.New(mt19937.New())
			rng.Seed(time.Now().UnixNano())
			target := 6

			if len(arg) >= 2 {
				if _target, e := strconv.ParseInt(arg[1], 10, 64); e != nil {
					return
				} else {
					target = int(_target)
				}
			} else {
				arg = append(arg, strconv.Itoa(target))
			}

			if target <= 1 {
				return
			}

			if target == 2 {
				str := ""
				if rng.Intn(target) == 0 {
					str = "앞면"
				} else {
					str = "뒷면"
				}
				sess.ChannelMessageSend(msg.ChannelID, fmt.Sprintln("동전을 던졌다:", str))
			} else {
				sess.ChannelMessageSend(msg.ChannelID, fmt.Sprintln(arg[1]+"면체 주사위를 굴렸다:", rng.Intn(target)+1))
			}
		},
	}

	result["pick"] = DiscordCommand{
		Description: "당첨시킨다. pick <A>...",
		Callback: func(arg []string, sess *discordgo.Session, msg *discordgo.Message) {
			rng := rand.New(mt19937.New())
			rng.Seed(time.Now().UnixNano())

			if len(arg) < 2 {
				members, e := sess.GuildMembers(msg.GuildID, "", 1000)
				if e != nil {
					return
				}

				for {
					target := members[rng.Intn(len(members))]
					if target.User.ID != sess.State.User.ID {
						sess.ChannelMessageSend(msg.ChannelID, fmt.Sprintln("여기 멤버 중 당첨자:", "`"+target.Nick+"`", "(`"+target.User.Username+"`)"))
						break
					}
				}
			} else {
				target := arg[1:][rng.Intn(len(arg[1:]))]
				sess.ChannelMessageSend(msg.ChannelID, fmt.Sprintln(arg[1:], "중 당첨자:", "`"+target+"`"))
			}
		},
	}

	result["eat"] = DiscordCommand{
		Description: "오늘 뭐 먹지",
		Callback: func(arg []string, sess *discordgo.Session, msg *discordgo.Message) {
			rng := rand.New(mt19937.New())
			rng.Seed(time.Now().UnixNano())

			list := []string{
				"치킨",
				"짜장면",
				"짬뽕",
				"탕수육",
				"피자",
				"볶음밥",
				"분식",
				"초밥+회",
				"족발",
				"보쌈",
				"쌀국수",
				"양꼬치",
				"설빙",
				"커피",
				"찜닭",
				"닭발",
				"곱창",
			}

			target := list[rng.Intn(len(list))]
			sess.ChannelMessageSend(msg.ChannelID, fmt.Sprintln("이거 먹어라:", "`"+target+"`"))
		},
	}

	result["??"] = DiscordCommand{
		Description: "해법 출력",
		Callback: func(arg []string, sess *discordgo.Session, msg *discordgo.Message) {
			temp := [][2]string{}
			for name, cmd := range result {
				temp = append(temp, [2]string{name, cmd.Description})
			}
			sort.Slice(temp, func(i, j int) bool {
				return temp[i][0] < temp[j][0]
			})
			content := ">>> "
			for _, cmd := range temp {
				content += "`" + cmd[0] + "`"
				content += " : "
				content += cmd[1]
				content += "\n"
			}
			sess.ChannelMessageSend(msg.ChannelID, content)
		},
	}

	return result
}
