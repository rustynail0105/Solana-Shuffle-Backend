package user

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/solanashuffle/backend/database"
	"github.com/solanashuffle/backend/env"
	"github.com/solanashuffle/backend/shuffle"
	"github.com/solanashuffle/backend/stream"
	"github.com/solanashuffle/backend/vsolana"

	"github.com/gagliardetto/solana-go"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

var (
	Chat *stream.Stream

	OnlinePlayers int = 3

	lastMessages []ChatPublicMessage
	antiSpamMap  = make(map[solana.PublicKey]struct{})

	slurRegex         = regexp.MustCompile(`pike?(ys?|ies)|pakis?|(ph|f)agg?s?([e0aio]ts?|oted|otry)|nigg?s?|nigg?[aeoi]s?|(ph|f)[@a]gs?|n[i!j1e]+gg?(rs?|ett?e?s?|lets?|ress?e?s?|r[a0oe]s?|[ie@ao0!]rs?|r[o0]ids?|ab[o0]s?|erest)|j[!i]gg?[aer]+(boo?s?|b00?s?)|jigg?[aer]+(b[0o]ing)|p[0o]rch\s*-?m[0o]nke?(ys?|ies?)|g(ooks?|00ks?)|k[iy]+kes?|b[ea]ne[ry]s?|(towel|rag)\s*heads?|wet\s*backs?|dark(e?y|ies?)|(shit|mud)\s*-?skins?|tarbab(ys?|ies?)|ape\s*-?fricans?|lesbos?|coons?(y|i?e?s?|er)|trann(ys?|ies?)|mignorants?|lady\s*-?boys?|spics?|r?coon\s*town|r?ni?1?ggers?|you\s*('?re|r)gay|shit\s*lords?|Homos?|groids?|chimpires?|mud\s*childr?e?n?|n[1!i]gs?-?|gays?(est|ly|er)|dune\s*coone?r?s?|high\s*yellows?|shee?\s*boons?|cock\s*suckers?|tards?|retards?|retard\*s?(ed|edly)|cunts?y?|dot\s*heads?|china\s*m[ae]n|queer\s*bags?|NAMBLA|fucking\s*(whores?)|puss(y|ies?)|ghey|whore\s*mouth|fuck\s*boys?|fat\s*fucks?|obeasts?|fuck\s*(wits?|tards?)|beetusbehemoths?|book\s*fags?|shit\s*(bags?|dicks?)|twats?|fupas?|holo\s*hoaxe?s?|Muslimes?|dind[ous]|boot\s*lips?|jig\s*apes?|nig\s*town|suspooks?`)
	antisemitismRegex = regexp.MustCompile(`J[3e]ws?|mein|kam[phf]|kram[phf]|hitler'?s?|Adolf'?s?|neo\s*nazis?`)
	suicideRegex      = regexp.MustCompile(`kill\s*your(self|selves)|commit\s*suicide|I\s*hope\s*(you|she|he)\s*dies?|kys`)
)

func init() {
	Chat = stream.New()
	go Chat.Start()
}

func SetWSGroup(group fiber.Router) {
	group.Use(HandleWSUpgrade)

	group.Get("/room/+", TransformRoomWSHandler(HandleRoomWS))
	group.Get("/chat", websocket.New(HandleChatWS))
	group.Get("/stats", websocket.New(HandleOnlinePlayersWS))
}

type ChatMessage struct {
	Type      string           `json:"type"`
	Value     string           `json:"value"`
	PublicKey solana.PublicKey `json:"publicKey,omitempty"`
	Signature solana.Signature `json:"signature"`

	Time int64 `json:"time"`
}

type ChatPublicMessage struct {
	Type      string           `json:"type"`
	Value     string           `json:"value"`
	PublicKey solana.PublicKey `json:"publicKey,omitempty"`
	Name      string           `json:"name"`
	Image     string           `json:"image"`

	Time int64 `json:"time"`
}

func HandleOnlinePlayersWS(c *websocket.Conn) {
	w := stream.NewWrapper(c)
	defer w.Conn.Close()

	/*
		defer func() {
			if r := recover(); r != nil {
				// Handle the error gracefully here
				log.Printf("Recovered from panic: %v", r)
			}
			w.Conn.Close()
		}
	*/

	closeChannel := make(chan error)

	err := w.WriteSafeJSON(fiber.Map{
		"type": "stats",
		"value": map[string]interface{}{
			"onlinePlayers": OnlinePlayers,
		},
	})
	if err != nil {
		return
	}

	go func() {
		ticker := time.NewTicker(time.Second * 5)
		for {
			select {
			case <-closeChannel:
				return
			case <-ticker.C:
				err := w.WriteSafeJSON(fiber.Map{
					"type": "stats",
					"value": map[string]interface{}{
						"onlinePlayers": OnlinePlayers,
					},
				})
				if err != nil {
					closeChannel <- err
				}
			}
		}
	}()

	for {
		_, _, err := w.Conn.ReadMessage()
		if err != nil {
			closeChannel <- err
			return
		}
	}
}

func HandleChatWS(c *websocket.Conn) {
	w := stream.NewWrapper(c)
	OnlinePlayers++

	defer func() {
		log.Println("unsubscribing")
		OnlinePlayers--
		Chat.UnsubscribeWebsocket(w)
		w.Conn.Close()
	}()

	Chat.SubscribeWebsocket(w)

	for _, msg := range lastMessages {
		err := w.WriteSafeJSON(msg)
		if err != nil {
			return
		}
	}

	for {
		var privateMsg ChatMessage
		err := c.ReadJSON(&privateMsg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println("read error")
			}
			log.Println("read err", err)
			return
		}

		if privateMsg.Type == "heartbeat" && privateMsg.Value == "ping" {
			err := w.WriteSafeJSON(fiber.Map{
				"type":  "heartbeat",
				"value": "pong",
			})
			if err != nil {
				log.Println("write error:", err)
				return
			}
		}

		if !vsolana.VerifySignature(privateMsg.Signature, privateMsg.PublicKey, []byte(fmt.Sprintf("solanashuffle chat %s", privateMsg.PublicKey.String()))) {
			continue
		}

		privateMsg.Time = time.Now().Unix()

		buf := bytes.Buffer{}
		var msg ChatPublicMessage
		if err := gob.NewEncoder(&buf).Encode(privateMsg); err != nil {
			return
		}
		gob.NewDecoder(&buf).Decode(&msg)

		if msg.Type == "message" {
			message := fmt.Sprint(msg.Value)
			if len(message) > 1000 || len(message) == 0 {
				continue
			}

			if slurRegex.MatchString(message) || antisemitismRegex.MatchString(message) || suicideRegex.MatchString(message) {
				w.WriteSafeJSON(ChatPublicMessage{
					Type:      "warning",
					Value:     "Please be appropriate in chat.",
					PublicKey: env.House().PublicKey(),
				})

				continue
			}

			if _, ok := antiSpamMap[msg.PublicKey]; ok {
				w.WriteSafeJSON(ChatPublicMessage{
					Type:      "warning",
					Value:     "Please slow down.",
					PublicKey: env.House().PublicKey(),
				})

				continue
			}

			user, err := database.DbGetUser(msg.PublicKey)
			if err == nil {
				msg.Name = user.Name
				msg.Image = user.Image
			}

			if len(lastMessages) >= 50 {
				lastMessages = append(lastMessages[1:], msg)
			} else {
				lastMessages = append(lastMessages, msg)
			}

			Chat.PublishJSON(msg)

			antiSpamMap[msg.PublicKey] = struct{}{}
			go func(pub solana.PublicKey) {
				time.Sleep(time.Second)
				delete(antiSpamMap, pub)
			}(msg.PublicKey)
		}
	}
}

func HandleRoomWS(room *shuffle.Room, c *websocket.Conn) {
	w := stream.NewWrapper(c)

	defer func() {
		log.Println("unsubscribing")
		room.UnsubscribeWebsocket(w)
		w.Conn.Close()
	}()

	room.SubscribeWebsocket(w)

	for {
		var msg ChatPublicMessage
		err := c.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println("read error")
			}
			log.Println("read err", err)
			return
		}

		if msg.Type == "heartbeat" && msg.Value == "ping" {
			err := w.WriteSafeJSON(fiber.Map{
				"type":  "heartbeat",
				"value": "pong",
			})
			if err != nil {
				log.Println("write error:", err)
				return
			}
		}
	}
}

func TransformRoomWSHandler(handler func(room *shuffle.Room, c *websocket.Conn)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("+")
		room, err := shuffle.GetRoom(id)
		if err != nil {
			return JSONError(c, fiber.StatusNotFound, err)
		}
		return websocket.New(func(c *websocket.Conn) {
			handler(room, c)
		})(c)
	}
}

func HandleWSUpgrade(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		c.Locals("allowed", true)
		return c.Next()
	}

	return fiber.ErrUpgradeRequired
}
