package price

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	http "github.com/bogdanfinn/fhttp"

	"github.com/gagliardetto/solana-go"
	"github.com/solanashuffle/backend/utility"
)

func EstimateCollectionPrice(collectionSymbol string) (int, error) {
	collectionStats, err := GetCollectionStats(collectionSymbol)
	if err != nil {
		return 0, err
	}

	if len(collectionStats) == 0 {
		limit := 1
		listeds, err := GetCollectionListed(collectionSymbol, limit)
		if err != nil {
			return 0, err
		}

		if len(listeds) != limit {
			return 0, err
		}

		currentFloorPrice := int(listeds[0].Price * float64(solana.LAMPORTS_PER_SOL))

		return currentFloorPrice, nil
	}

	var validFloors []int

	if len(collectionStats) < 6 {
		for _, stats := range collectionStats[0 : len(collectionStats)-1] {
			validFloors = append(validFloors, int(stats.CurrentFloorPrice*float64(solana.LAMPORTS_PER_SOL)))
		}
	} else {
		for _, stats := range collectionStats[len(collectionStats)-6:] {
			validFloors = append(validFloors, int(stats.CurrentFloorPrice*float64(solana.LAMPORTS_PER_SOL)))
		}
	}

	var sum int

	for _, floor := range validFloors {
		sum += floor
	}

	avg := int(float64(sum) / (float64(len(validFloors))))

	return avg, nil
}

type collectionStatsResponse []struct {
	CurrentFloorPrice  float64 `json:"cFP"`
	CurrentListedCount int     `json:"cLC"`
	CurrentVolume      float64 `json:"cV"`
	MaxFloorPrice      float64 `json:"maxFP"`
	MinFloorPrice      float64 `json:"minFP"`
	LastFloorPrice     float64 `json:"oFP"`
	LastListedCount    int     `json:"oLC"`
	LastVolume         float64 `json:"oV"`
	Time               int64   `json:"ts"`
}

func GetCollectionStats(symbol string) (collectionStatsResponse, error) {
	client, err := utility.NewTLSClient()
	if err != nil {
		return collectionStatsResponse{}, err
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/collection_stats/getCollectionTimeSeries/%s?edge_cache=true&resolution=6h&addLastDatum=true", MAGICEDENSTATS, symbol), nil)
	if err != nil {
		return collectionStatsResponse{}, err
	}

	req.Header = utility.BrowserHeaders()

	resp, err := client.Do(req)
	if err != nil {
		return collectionStatsResponse{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return collectionStatsResponse{}, err
	}

	var parsed collectionStatsResponse
	err = json.Unmarshal(body, &parsed)
	if err != nil {
		return collectionStatsResponse{}, err
	}

	return parsed, nil
}

type listedResponse struct {
	Results []listed `json:"results"`
}

type listed struct {
	Price float64 `json:"price"`
}

func GetCollectionListed(symbol string, limit int) ([]listed, error) {
	client, err := utility.NewTLSClient()
	if err != nil {
		return []listed{}, err
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/getListedNftsByCollectionSymbol?collection=%s&direction=2&field=1&limit=%d&offset=0", MAGICEDENIDXV2, symbol, limit), nil)
	if err != nil {
		return []listed{}, err
	}

	req.Header = utility.BrowserHeaders()

	resp, err := client.Do(req)
	if err != nil {
		return []listed{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []listed{}, err
	}

	var parsed listedResponse
	err = json.Unmarshal(body, &parsed)
	if err != nil {
		return []listed{}, err
	}

	return parsed.Results, nil
}
