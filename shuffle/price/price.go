package price

// estimates the price of nfts based on 24h ME floor price
// price uses redis for extra speedy estimates
// code fully works, no need to change, not the prettiest though

import (
	"encoding/binary"
	"errors"
	"sync"
	"time"

	"github.com/solanashuffle/backend/stream"
)

const (
	MAGICEDENRPC   = "https://api-mainnet.magiceden.io/rpc"
	MAGICEDENSTATS = "https://stats-mainnet.magiceden.io"
	MAGICEDENIDXV2 = "https://api-mainnet.magiceden.io/idxv2"
)

var (
	collectionMapMutex      = sync.RWMutex{}
	collectionProcessingMap = make(map[string]*stream.Stream)

	priceCache = make(map[string]int)
	priceMutex = sync.RWMutex{}
)

func init() {
	go Routine()
}

func Routine() {
	for {
		time.Sleep(time.Minute * 10)
		clearPriceCache()
	}
}

func Estimate(collectionSymbol string) (int, error) {
	if p, err := getPriceCache(collectionSymbol); err == nil {
		return p, nil
	}

	collectionMapMutex.RLock()
	s, ok := collectionProcessingMap[collectionSymbol]
	collectionMapMutex.RUnlock()
	if ok {
		ch := s.Subscribe()
		defer s.Unsubscribe(ch)
		priceBytes := <-ch
		price := binary.BigEndian.Uint64(priceBytes)
		if price == 0 {
			return 0, errors.New("could not get price")
		}
		return int(price), nil
	}

	collectionMapMutex.Lock()
	priceStream := stream.New()
	go priceStream.Start()
	collectionProcessingMap[collectionSymbol] = priceStream
	collectionMapMutex.Unlock()
	price, err := EstimateCollectionPrice(collectionSymbol)
	if err == nil {
		setPriceCache(collectionSymbol, price)
	}
	priceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(priceBytes, uint64(price))
	priceStream.Publish(priceBytes)
	collectionMapMutex.Lock()
	delete(collectionProcessingMap, collectionSymbol)
	collectionMapMutex.Unlock()

	return price, err
}

func clearPriceCache() {
	priceCache = make(map[string]int)
}

func setPriceCache(key string, val int) {
	priceMutex.Lock()
	priceCache[key] = val
	priceMutex.Unlock()
}

func getPriceCache(key string) (int, error) {
	priceMutex.RLock()
	fp, ok := priceCache[key]
	priceMutex.RUnlock()
	if !ok {
		return 0, errors.New("not found")
	}

	return fp, nil
}
