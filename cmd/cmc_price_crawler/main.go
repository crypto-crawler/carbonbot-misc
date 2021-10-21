package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	"github.com/gorilla/websocket"
	"github.com/soulmachine/coinsignal/config"
	"github.com/soulmachine/coinsignal/pojo"
	"github.com/soulmachine/coinsignal/pubsub"
	"github.com/soulmachine/coinsignal/utils"
)

// CoinMarketCap top cryptocurrencies
func fetch_cmc_top(limit int) map[int64]string {
	url := fmt.Sprintf("https://api.coinmarketcap.com/data-api/v3/cryptocurrency/listing?start=1&limit=%v&sortBy=market_cap&sortType=desc&convert=USD&cryptoType=all&tagType=all&audited=false", limit)
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(req)
	if err != nil {
		panic(err.Error())
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	m := make(map[int64]string)
	jsonparser.ArrayEach(body, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		id, err := jsonparser.GetInt(value, "id")
		if err != nil {
			panic(err)
		}
		symbol, err := jsonparser.GetString(value, "symbol")
		if err != nil {
			panic(err)
		}
		m[id] = symbol
	}, "data", "cryptoCurrencyList")
	return m
}

func min(a, b int) int {
	if a <= b {
		return a
	} else {
		return b
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
		rf = utils.NewRollingFile(data_dir, "cmc.prices")
	}

	var publisher *pubsub.Publisher
	if len(redis_url) == 0 {
		publisher = nil
		log.Println("The REDIS_URL environment variable is empty")
	} else {
		utils.WaitRedis(ctx, redis_url)
		publisher = pubsub.NewPublisher(ctx, redis_url)
	}

	client, _, err := websocket.DefaultDialer.Dial("wss://stream.coinmarketcap.com/price/latest", nil)
	if err != nil {
		log.Fatal(err)
	}

	currencyMap := fetch_cmc_top(5000)
	currencyIds := make([]int64, 0, len(currencyMap))
	for id := range currencyMap {
		currencyIds = append(currencyIds, id)
	}

	chunk_size := 100
	for i := 0; i < len(currencyIds); i += chunk_size {
		chunk := currencyIds[i:min(i+chunk_size, len(currencyIds))]
		command := fmt.Sprintf("{\"method\":\"subscribe\",\"id\":\"price\",\"data\":{\"cryptoIds\":%s,\"index\":null}}", strings.Join(strings.Split(fmt.Sprint(chunk), " "), ","))
		// command := "{\"method\":\"subscribe\",\"id\":\"price\",\"data\":{\"cryptoIds\":[1],\"index\":null}}"
		err = client.WriteMessage(websocket.TextMessage, []byte(command))
		if err != nil {
			log.Fatalln("Subscription failed: ", err)
		}
	}

	for {
		_, json_bytes, err := client.ReadMessage()
		if err != nil {
			panic(err)
		}
		if rf != nil {
			rf.Write(string(json_bytes) + "\n")
		}
		idStr, _, _, _ := jsonparser.Get(json_bytes, "d", "cr", "id")
		priceStr, _, _, _ := jsonparser.Get(json_bytes, "d", "cr", "p")

		id, _ := strconv.ParseInt(string(idStr), 0, 64)
		currency, ok := currencyMap[id]
		if !ok {
			// log.Println("Failed to find symbol for id ", id)
			continue
		}
		price, _ := strconv.ParseFloat(string(priceStr), 64)

		currency_price := &pojo.CurrencyPrice{
			Currency: currency,
			Price:    price,
		}
		json_bytes, _ = json.Marshal(currency_price)
		if publisher != nil {
			publisher.Publish(config.REDIS_TOPIC_CURRENCY_PRICE_CHANNEL, string(json_bytes))
		}
	}

	// rf.Close()
	// publisher.Close()
}
