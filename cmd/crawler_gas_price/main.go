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

func fetch_gas_price(rf *utils.RollingFile) *GasPriceMsg {
	url := "https://etherchain.org/api/gasnow"
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)

	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	if rf != nil {
		rf.Write(string(body) + "\n")
	}

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
	redis_url := os.Getenv("REDIS_URL")
	if len(data_dir) == 0 && len(redis_url) == 0 {
		log.Fatal("Both DATA_DIR and REDIS_URL are empty")
	}

	var rf *utils.RollingFile
	if len(data_dir) == 0 {
		log.Println("The DATA_DIR environment variable is empty")
		rf = nil
	} else {
		rf = utils.NewRollingFile(data_dir, "gasnow.gas_price")
	}

	var publisher *pubsub.Publisher
	if len(redis_url) == 0 {
		publisher = nil
		log.Println("The REDIS_URL environment variable is empty")
	} else {
		utils.WaitRedis(ctx, redis_url)
		publisher = pubsub.NewPublisher(ctx, redis_url)
	}

	ticker := time.NewTicker(5 * time.Second) // check every 15 seconds
	defer ticker.Stop()

	for range ticker.C {
		gas_price := fetch_gas_price(rf)
		if gas_price != nil {
			bytes, err := json.Marshal(gas_price)
			if err != nil {
				panic(err)
			} else {
				if publisher != nil {
					publisher.Publish(config.REDIS_TOPIC_ETH_GAS_PRICE, string(bytes))
				}
			}
		}
	}

	if rf != nil {
		rf.Close()
	}
	if publisher != nil {
		publisher.Close()
	}
}
