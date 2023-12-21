package shuffle

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gofiber/fiber/v2"
	"github.com/solanashuffle/backend/database"
	"github.com/solanashuffle/backend/stream"
	"github.com/solanashuffle/backend/utility"
)

const (
	tickDelay        = time.Second * 1
	DefaultCountdown = time.Second * 40
)

func (r *Room) Routine() {
	maxCountdown := DefaultCountdown
	if r.Name == "24h Lotto" && r.Official {
		maxCountdown = time.Hour * 24
	}
	go func() {
		for {
			r.NewSession()
			fmt.Println("init new session")
			r.Session.run(r.stream, r.Token, maxCountdown)
		}
	}()
}

func (s *Session) run(stream *stream.Stream, token database.Token, maxCountdown time.Duration) {
	s.Status = "waiting"
	s.Countdown = maxCountdown

	stream.PublishJSON(fiber.Map{
		"type":   "reset",
		"value":  int(s.Countdown),
		"status": s.Status,
		"id":     s.ID,
	})

	s.WaitUntilPopulated(stream)

	go func() {
		sub := stream.Subscribe()
		defer stream.Unsubscribe(sub)
		for {
			j := <-sub
			msg := make(fiber.Map)
			json.Unmarshal(j, &msg)
			if msg["type"] == "waitingSolana" {
				return
			}
			if msg["type"] != "newUser" {
				continue
			}
			if s.Countdown > time.Second*10 {
				continue
			}
			s.Countdown = time.Second * 11
		}
	}()

	if maxCountdown == time.Hour*24 {
		today := utility.FormatDate(time.Now())
		todayT, err := utility.ParseDate(today)
		if err != nil {
			panic(err)
		}
		fmt.Println(todayT)
		todayT = todayT.AddDate(0, 0, 1)
		fmt.Println(todayT)
		s.Countdown = time.Until(todayT).Round(time.Second)
	}

	for _ = 0; s.Countdown >= 0; s.Countdown -= time.Second {
		stream.PublishJSON(fiber.Map{
			"type":   "waitingCountdown",
			"value":  int(s.Countdown),
			"status": s.Status,
			"id":     s.ID,
		})
		time.Sleep(tickDelay)
	}

	s.Countdown = 0

	s.Status = "waitingSolana"

	stream.PublishJSON(fiber.Map{
		"type":   "waitingSolana",
		"status": s.Status,
		"id":     s.ID,
	})

	s.WaitUntilNotOnHold()
	s.Status = "drawing"
	result, resultAnnotationChannel, err := s.draw(token)
	if err != nil {
		stream.PublishJSON(fiber.Map{
			"type":   "error",
			"value":  "Please contact support",
			"status": s.Status,
			"id":     s.ID,
		})
		return
	}
	stream.PublishJSON(fiber.Map{
		"type":   "result",
		"value":  result,
		"status": s.Status,
		"id":     s.ID,
	})
	go func(s Session, resultAnnotationChannel chan []solana.Signature) {
		buf := bytes.Buffer{}
		if err := gob.NewEncoder(&buf).Encode(s); err != nil {
			return
		}
		gob.NewDecoder(&buf).Decode(&s)
		signatures := <-resultAnnotationChannel
		s.Result.Signatures = signatures
		stream.PublishJSON(fiber.Map{
			"type":   "resultAnnotation",
			"value":  signatures,
			"status": s.Status,
			"id":     s.ID,
		})

		err := s.Track(token)
		log.Println("tracked", err)
	}(*s, resultAnnotationChannel)

	time.Sleep(time.Second * 10)
	s.Status = "finished"
	for i := time.Second * 5; i > 0; i -= time.Second {
		s.Countdown = i
		stream.PublishJSON(fiber.Map{
			"type":   "resetCountdown",
			"value":  int(i),
			"status": s.Status,
			"id":     s.ID,
		},
		)
		time.Sleep(tickDelay)
	}
}
