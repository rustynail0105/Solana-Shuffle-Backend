package price

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	http "github.com/bogdanfinn/fhttp"
	"github.com/gagliardetto/solana-go"
	"github.com/solanashuffle/backend/database"
	"github.com/solanashuffle/backend/utility"
)

type mintListResponse struct {
	Results []struct {
		MintAddress solana.PublicKey `json:"mintAddress"`
	} `json:"results"`
}

func WhitelistCollection(collectionSymbol string, hadeswapMarket solana.PublicKey) error {
	var mintList []solana.PublicKey

	j, err := ioutil.ReadFile(fmt.Sprintf("./%s.json", collectionSymbol))
	if err != nil {
		mintList, err = GetCollectionMintList(collectionSymbol)
		if err != nil {
			return err
		}
	} else {
		err = json.Unmarshal(j, &mintList)
		if err != nil {
			mintList, err = GetCollectionMintList(collectionSymbol)
			if err != nil {
				return err
			}
		}
	}

	err = database.RDB.Set(
		context.TODO(),
		collectionSymbol,
		hadeswapMarket.String(),
		0,
	).Err()
	if err != nil {
		return err
	}

	chunks := utility.ChunkBy(mintList, 50)

	for _, chunk := range chunks {
		setArr := []string{}
		for _, pub := range chunk {
			setArr = append(setArr, pub.String())
			setArr = append(setArr, collectionSymbol)
		}
		err := database.RDB.MSet(
			context.TODO(),
			setArr,
		).Err()
		if err != nil {
			return err
		}
	}

	return nil
}

func GetCollectionMintList(collectionSymbol string) ([]solana.PublicKey, error) {
	var mintList []solana.PublicKey

	client, err := utility.NewTLSClient()
	if err != nil {
		return []solana.PublicKey{}, err
	}

	limit := 100
	offset := 0
	for size := 100; size > 0; {
		fmt.Println(size, offset)
		url := fmt.Sprintf(
			"%s/getAllNftsByCollectionSymbol?collectionSymbol=%s&direction=1&field=1&limit=%d&offset=%d",
			MAGICEDENIDXV2,
			collectionSymbol,
			limit,
			offset,
		)
		offset += 100

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return []solana.PublicKey{}, err
		}

		req.Header = utility.BrowserHeaders()

		resp, err := client.Do(req)
		if err != nil {
			return []solana.PublicKey{}, err
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return []solana.PublicKey{}, err
		}

		var parsed mintListResponse
		err = json.Unmarshal(body, &parsed)
		if err != nil {
			return []solana.PublicKey{}, err
		}

		for _, res := range parsed.Results {
			mintList = append(mintList, res.MintAddress)
		}

		size = len(parsed.Results)
	}

	return mintList, nil
}
