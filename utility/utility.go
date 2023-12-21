package utility

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/solanashuffle/backend/env"
)

var (
	False      = false
	TimeLayout = "2006-01-02"
)

func SendWebhook(amount int) {
	embed := discord.NewEmbedBuilder().
		SetTitle("New Bet")

	embed.AddField("Type", "Sol", false)

	solAmount := amount / 1000000000

	embed.AddField("Value", fmt.Sprintf("%v SOL", solAmount), false)

	data, err := json.Marshal(discord.MessageCreate{Embeds: []discord.Embed{embed.Build()}})
	if err != nil {
		log.Printf("Failed to marshal webhook data %q", err)
		return
	}

	webhook := env.GetWebhook()

	_, err = http.Post(webhook, "application/json", bytes.NewReader(data))

	if err != nil {
		log.Printf("Failed to send webhook %q", err)
		return
	}
}

func ChunkBy[T any](items []T, chunkSize int) (chunks [][]T) {
	var _chunks = make([][]T, 0, (len(items)/chunkSize)+1)
	for chunkSize < len(items) {
		items, _chunks = items[chunkSize:], append(_chunks, items[0:chunkSize:chunkSize])
	}
	return append(_chunks, items)
}

func Remove[T any](slice []T, s int) []T {
	return append(slice[:s], slice[s+1:]...)
}

func ContainsInt(arr []int, i int) bool {
	for _, e := range arr {
		if e == i {
			return true
		}
	}
	return false
}

func MergeChannels[T any](channels ...chan T) chan T {
	mergeCh := make(chan T)

	var wg sync.WaitGroup
	wg.Add(len(channels))

	for _, ch := range channels {
		go func(ch chan T) {
			for msg := range ch {
				mergeCh <- msg
			}
			wg.Done()
		}(ch)
	}

	go func() {
		wg.Wait()
		close(mergeCh)
	}()

	return mergeCh
}

func RandomInt(min, max int) int {
	max++
	bg := big.NewInt(int64(max - min))

	n, err := rand.Int(rand.Reader, bg)
	if err != nil {
		return min
	}

	return int(n.Int64() + int64(min))
}

func Chance(basisPoints int) bool {
	rand := RandomInt(1, 10000)
	return rand <= basisPoints
}

func FormatDate(t time.Time) string {
	return t.Format(TimeLayout)
}

func ParseDate(d string) (time.Time, error) {
	return time.Parse(TimeLayout, d)
}
