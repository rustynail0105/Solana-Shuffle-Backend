package fair

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"math"
	"strconv"
	"sync"

	"github.com/solanashuffle/backend/utility"
)

// Client Provably fair client
type Client struct {
	ServerSeed []byte
	Nonce      uint64

	mux sync.Mutex
}

// Generate new number between 0 and n. Returns (new number, serverSeed, nonce, error)
func (c *Client) Generate(clientSeed []byte, n int) (float64, []byte, uint64, error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if c.Nonce == math.MaxUint64 {
		newSeed, err := GenerateNewSeed(len(c.ServerSeed))
		if err != nil {
			return 0, nil, 0, err
		}
		c.ServerSeed = newSeed
		c.Nonce = 0
	}
	hmacBytes := c.getHMACString(clientSeed)
	hmacStr := string(hmacBytes)

	var randNum uint64
	var err error
	for i := 0; i < len(hmacStr)-5; i++ {
		// Get the index for this segment and ensure it doesn't overrun the slice
		idx := i * 5
		if len(hmacStr) < (idx + 5) {
			break
		}

		// Get 5 characters and convert them to decimal
		randNum, err = strconv.ParseUint(hmacStr[idx:idx+5], 16, 0)
		if err != nil {
			return 0, nil, 0, err
		}

		// Continue unless our number was greater than our max
		if randNum <= 999999 {
			break
		}
	}

	// If even the last segment was invalid we must give up
	if randNum > 999999 {
		return 0, nil, 0, errors.New("invalid nonce")
	}

	c.Nonce++
	// Normalize the number to [0,n]
	return float64(randNum%10000) / float64(n), c.ServerSeed, c.Nonce - 1, nil
}

// GenerateFromString generate new number from hex string
func (c *Client) GenerateFromString(clientSeed string, n int) (float64, []byte, uint64, error) {
	seed, err := hex.DecodeString(clientSeed)
	if err != nil {
		return 0, nil, 0, err
	}
	return c.Generate(seed, n)
}

// GenerateNewSeed generate new seed
func GenerateNewSeed(byteCount int) ([]byte, error) {
	seed := make([]byte, byteCount)
	_, err := rand.Read(seed)
	return seed, err
}

func (c *Client) getHMACString(clientSeed []byte) []byte {
	h := hmac.New(sha512.New, c.ServerSeed)
	h.Write(append(append(clientSeed, '-'), []byte(strconv.FormatUint(c.Nonce, 10))...))

	hmacBytes := make([]byte, 128)
	hex.Encode(hmacBytes, h.Sum(nil))
	return hmacBytes
}

// Verify takes a state and checks that the supplied number was fairly generated
func Verify(clientSeed []byte, serverSeed []byte, nonce uint64, randNum float64, n int) (bool, error) {
	client := &Client{
		ServerSeed: serverSeed,
		Nonce:      nonce,
	}

	num, _, _, err := client.Generate(clientSeed, n)
	if err != nil {
		return false, err
	}

	return num == randNum, nil
}

// VerifyFromString verify from string clientSeed and serverSeed
func VerifyFromString(clientSeed, serverSeed string, nonce uint64, randNum float64, n int) (bool, error) {
	clientSeedBytes, err := hex.DecodeString(clientSeed)
	if err != nil {
		return false, err
	}
	serverSeedBytes, err := hex.DecodeString(serverSeed)
	if err != nil {
		return false, err
	}
	return Verify(clientSeedBytes, serverSeedBytes, nonce, randNum, n)
}

func RandomUniqueIntArray(size, min, max int) []int {
	var arr []int
	if size <= 0 {
		return arr
	}

	for i := 0; i < size; i++ {
		r := utility.RandomInt(min, max)
		for utility.ContainsInt(arr, r) {
			r = utility.RandomInt(min, max)
		}
		arr = append(arr, r)
	}

	return arr
}
