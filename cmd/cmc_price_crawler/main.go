package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/buger/jsonparser"
	"github.com/gorilla/websocket"
	"github.com/soulmachine/coinsignal/config"
	"github.com/soulmachine/coinsignal/pojo"
	"github.com/soulmachine/coinsignal/pubsub"
	"github.com/soulmachine/coinsignal/utils"
)

type currencyId struct {
	Id       int64
	Currency string
}

// CoinMarketCap top cryptocurrencies
func fetch_cmc_top(limit int) []currencyId {
	url := fmt.Sprintf("https://api.coinmarketcap.com/data-api/v3/cryptocurrency/listing?start=1&limit=%v&sortBy=market_cap&sortType=desc&convert=USD&cryptoType=all&tagType=all&audited=false", limit)
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(req)
	if err != nil {
		panic(err.Error())
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	arr := make([]currencyId, 0)
	jsonparser.ArrayEach(body, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		id, err := jsonparser.GetInt(value, "id")
		if err != nil {
			panic(err)
		}
		symbol, err := jsonparser.GetString(value, "symbol")
		if err != nil {
			panic(err)
		}
		arr = append(arr, currencyId{Id: id, Currency: symbol})
	}, "data", "cryptoCurrencyList")
	return arr
}

func subscribe_ids(ids []int64, stopCh <-chan struct{}, outCh chan<- []byte) {
	client, _, err := websocket.DefaultDialer.Dial("wss://stream.coinmarketcap.com/price/latest", nil)
	if err != nil {
		log.Fatal(err)
	}

	command := fmt.Sprintf("{\"method\":\"subscribe\",\"id\":\"price\",\"data\":{\"cryptoIds\":%s,\"index\":null}}", strings.Join(strings.Split(fmt.Sprint(ids), " "), ","))
	// command := "{\"method\":\"subscribe\",\"id\":\"price\",\"data\":{\"cryptoIds\":[1],\"index\":null}}"
	err = client.WriteMessage(websocket.TextMessage, []byte(command))
	if err != nil {
		log.Fatalln("Subscription failed: ", err)
	}

	go func() {
		for {
			select {
			case <-stopCh:
				client.Close()
				return
			default:
				_, json_bytes, err := client.ReadMessage()
				if err != nil {
					panic(err)
				}
				outCh <- json_bytes
			}
		}
	}()
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

	// catch Ctrl+C
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	stopCh := make(chan struct{})

	msgCh := make(chan []byte)

	currencyMap := make(map[int64]string)
	currencyIds := fetch_cmc_top(5000)
	for _, x := range currencyIds {
		currencyMap[x.Id] = x.Currency
	}

	chunk_size := 2000
	for i := 0; i < len(currencyIds); i += chunk_size {
		chunk := currencyIds[i:min(i+chunk_size, len(currencyIds))]
		ids := make([]int64, 0)
		for _, id := range chunk {
			ids = append(ids, id.Id)
		}
		subscribe_ids(ids, stopCh, msgCh)
	}

	for {
		select {
		case <-signals:
			log.Println("Ctrl+C detected, exiting...")
			close(stopCh)
			time.Sleep(time.Second) // give some time for other goroutines to stop
			return
		case json_bytes := <-msgCh:
			idStr, _, _, _ := jsonparser.Get(json_bytes, "d", "cr", "id")
			priceStr, _, _, _ := jsonparser.Get(json_bytes, "d", "cr", "p")

			id, _ := strconv.ParseInt(string(idStr), 0, 64)
			currency, ok := currencyMap[id]
			if !ok {
				// log.Println("Failed to find symbol for id ", id)
				break
			}

			// Add currency
			json_bytes, _ = jsonparser.Set(json_bytes, []byte("\""+currency+"\""), "d", "cr", "c")
			if rf != nil {
				rf.Write(string(json_bytes) + "\n")
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
	}

	// rf.Close()
	// publisher.Close()
}
