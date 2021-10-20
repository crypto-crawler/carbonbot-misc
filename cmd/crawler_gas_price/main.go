package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/buger/jsonparser"
	"github.com/soulmachine/coinsignal/config"
	"github.com/soulmachine/coinsignal/pubsub"
	"github.com/soulmachine/coinsignal/utils"
)

// price in Wei
type GasPriceMsg struct {
	Rapid     uint64  `json:"rapid"`
	Fast      uint64  `json:"fast"`
	Standard  uint64  `json:"standard"`
	Slow      uint64  `json:"slow"`
	Timestamp int64   `json:"timestamp"`
	PriceUSD  float64 `json:"priceUSD"`
}

// wei to USD
func fromWei(wei uint64, eth_price float64) float64 {
	return float64(wei) / 1000000000000000000 * 21000 * eth_price
}

func fetch_gas_price() *GasPriceMsg {
	url := "https://etherchain.org/api/gasnow"
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)

	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	data, _, _, err := jsonparser.Get(body, "data")
	if err != nil {
		log.Println(err)
		return nil
	}

	msg := GasPriceMsg{}
	err = json.Unmarshal(data, &msg)
	if err != nil {
		log.Println(err)
		return nil
	} else {
		return &msg
	}
}

func main() {
	ctx := context.Background()

	data_dir := os.Getenv("DATA_DIR")
	var rf *utils.RollingFile
	if len(data_dir) == 0 {
		log.Println("The DATA_DIR environment variable is empty")
		rf = nil
	} else {
		rf = utils.NewRollingFile(data_dir, "gasnow.gas_price")
	}

	redis_url := os.Getenv("REDIS_URL")
	if len(redis_url) == 0 {
		log.Println("The REDIS_URL environment variable is empty")
	} else {
		utils.WaitRedis(ctx, redis_url)
	}
	publisher := pubsub.NewPublisher(ctx, redis_url)

	ticker := time.NewTicker(5 * time.Second) // check every 15 seconds
	defer ticker.Stop()

	for range ticker.C {
		gas_price := fetch_gas_price()
		if gas_price != nil {
			bytes, err := json.Marshal(gas_price)
			if err != nil {
				panic(err)
			} else {
				publisher.Publish(config.REDIS_TOPIC_ETH_GAS_PRICE, string(bytes))
				if rf != nil {
					rf.Write(string(bytes) + "\n")
				}
			}
		}
	}

	rf.Close()
	publisher.Close()
}
