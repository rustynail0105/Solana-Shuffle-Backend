package conversion

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/solanashuffle/backend/database"
	"github.com/solanashuffle/backend/utility"
	"go.mongodb.org/mongo-driver/bson"
)

var (
	solCache = make(map[solana.PublicKey]float64)
	endpoint = "https://price.jup.ag/v3"
)

type jupiterResponse struct {
	Data map[solana.PublicKey]struct {
		ID            string  `json:"id"`
		Mintsymbol    string  `json:"mintSymbol"`
		Vstoken       string  `json:"vsToken"`
		Vstokensymbol string  `json:"vsTokenSymbol"`
		Price         float64 `json:"price"`
	} `json:"data"`
	Timetaken   float64 `json:"timeTaken"`
	Contextslot int     `json:"contextSlot"`
}

func Routine() {
	go routine()
}

func routine() {
	for {
		var tokens []database.Token
		err := database.Find("tokens", bson.M{}, &tokens)
		if err != nil {
			continue
		}

		var removeIndeces []int
		for i, token := range tokens {
			if token.Ticker == "SOL" || token.Ticker == "IOS" {
				removeIndeces = append(removeIndeces, i)
			}
		}

		for i, s := range removeIndeces {
			tokens = utility.Remove(tokens, s-i)
		}

		var args []string
		for _, token := range tokens {
			args = append(args, token.PublicKey.String())
		}
		url := fmt.Sprintf("%s/price?ids=%s&vsToken=SOL", endpoint, strings.Join(args, ","))
		resp, err := http.Get(url)
		if err != nil {
			continue
		}

		b, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}
		var parsed jupiterResponse
		err = json.Unmarshal(b, &parsed)
		if err != nil {
			continue
		}

		if len(parsed.Data) != len(args) {
			continue
		}

		for _, token := range tokens {
			data, ok := parsed.Data[token.PublicKey]
			if !ok {
				continue
			}
			solCache[token.PublicKey] = data.Price
		}

		time.Sleep(time.Minute)
	}
}

func ToSOL(amount int, mint solana.PublicKey) (int, error) {
	if mint == solana.SolMint {
		return amount, nil
	}

	multiplier, ok := solCache[mint]
	if !ok {
		return 0, errors.New("token not found")
	}

	return int(float64(amount) * multiplier), nil
}
