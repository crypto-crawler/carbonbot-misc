package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	"github.com/soulmachine/coinsignal/config"
	"github.com/soulmachine/coinsignal/pojo"
	"github.com/soulmachine/coinsignal/pubsub"
	"github.com/soulmachine/coinsignal/utils"
)

// Get spot currency prices, mainly for fiat currencies
func fetchFtxMarkets(rf *utils.RollingFile) map[string]float64 {
	url := "https://ftx.com/api/markets"
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)

	resp, err := client.Do(req)
	if err != nil {
		panic(err.Error())
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return map[string]float64{}
	}
	result, _, _, err := jsonparser.Get(body, "result")
	if err != nil {
		log.Println(err)
		return map[string]float64{}
	}
	if rf != nil {
		rf.Write(string(body) + "\n")
	}

	var mapping = make(map[string]float64)

	jsonparser.ArrayEach(result, func(element []byte, dataType jsonparser.ValueType, offset int, err error) {
		market_type, _, _, _ := jsonparser.Get(element, "type")
		if string(market_type) == "spot" {
			name, _, _, _ := jsonparser.Get(element, "name")
			arr := strings.Split(string(name), "/")
			base := arr[0]
			quote := arr[1]
			if quote == "USD" || quote == "USDT" || quote == "USDC" || quote == "BUSD" {
				price, _, _, _ := jsonparser.Get(element, "price")
				price_float, _ := strconv.ParseFloat(string(price), 64)
				mapping[base] = price_float
			}
		}
	})
	return mapping
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
		rf = utils.NewRollingFile(data_dir, "ftx.markets")
	}

	var publisher *pubsub.Publisher
	if len(redis_url) == 0 {
		publisher = nil
		log.Println("The REDIS_URL environment variable is empty")
	} else {
		utils.WaitRedis(ctx, redis_url)
		publisher = pubsub.NewPublisher(ctx, redis_url)
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		mapping := fetchFtxMarkets(rf)
		for currency, price := range mapping {
			currency_price := pojo.CurrencyPrice{
				Currency: currency,
				Price:    price,
			}
			json_bytes, _ := json.Marshal(currency_price)

			if publisher != nil {
				publisher.Publish(config.REDIS_TOPIC_CURRENCY_PRICE_CHANNEL, string(json_bytes))
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
