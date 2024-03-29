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
	"time"

	"github.com/buger/jsonparser"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/soulmachine/coinsignal/config"
	"github.com/soulmachine/coinsignal/pojo"
	"github.com/soulmachine/coinsignal/pubsub"
	"github.com/soulmachine/coinsignal/utils"
)

// return ETH number
func fetchBlockReward(blockNumber int64) float64 {
	etherscan_api_key := os.Getenv("ETHERSCAN_API_KEY")
	url := fmt.Sprintf("https://api.etherscan.io/api?module=block&action=getblockreward&blockno=%d&apikey=%s", blockNumber, etherscan_api_key)

	for i := 0; i < 3; i++ {
		time.Sleep(5 * time.Second)
		resp, err := http.Get(url)
		if err != nil {
			log.Fatalln(err)
		}
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)

		message, _, _, _ := jsonparser.Get(body, "message")
		blockRewardStr, _, _, _ := jsonparser.Get(body, "result", "blockReward")
		if string(message) == "OK" {
			blockReward, _ := strconv.ParseUint(string(blockRewardStr), 10, 64)
			return float64(blockReward) / 1000000000000000000
		}
	}

	return 2.25926 // default value, see https://bitinfocharts.com/ethereum/
}

func main() {
	ctx := context.Background()

	full_node_url := os.Getenv("FULL_NODE_URL")
	if len(full_node_url) == 0 {
		log.Fatal("The FULL_NODE_URL environment variable is empty")
	}
	if len(os.Getenv("ETHERSCAN_API_KEY")) == 0 {
		log.Fatal("The ETHERSCAN_API_KEY environment variable is empty")
	}

	data_dir := os.Getenv("DATA_DIR")
	var rf *utils.RollingFile
	if len(data_dir) == 0 {
		log.Println("The DATA_DIR environment variable is empty")
		rf = nil
	} else {
		rf = utils.NewRollingFile(data_dir, "eth.block_header")
	}

	redis_url := os.Getenv("REDIS_URL")
	if len(redis_url) == 0 {
		log.Fatal("The REDIS_URL environment variable is empty")
	}
	utils.WaitRedis(ctx, redis_url)
	rdb := utils.NewRedisClient(redis_url)

	client, err := ethclient.Dial(full_node_url)
	if err != nil {
		log.Fatal(err)
	}

	headers := make(chan *types.Header)
	sub, err := client.SubscribeNewHead(context.Background(), headers)
	if err != nil {
		log.Fatal(err)
	}

	publisher := pubsub.NewPublisher(ctx, redis_url)
	pubsub := rdb.Subscribe(ctx,
		config.REDIS_TOPIC_CURRENCY_PRICE_CHANNEL,
	)
	var ethPrice float64
	for {
		select {
		case err := <-sub.Err():
			log.Fatal(err)
		case msg := <-pubsub.Channel():
			currency_price := pojo.CurrencyPrice{}
			json.Unmarshal([]byte(msg.Payload), &currency_price)
			if currency_price.Currency == "ETH" {
				ethPrice = currency_price.Price
			}
		case header := <-headers:
			json_bytes, _ := json.Marshal(header)

			blockNumberBytes, _, _, _ := jsonparser.Get(json_bytes, "number")
			blockNumber, _ := strconv.ParseInt(string(blockNumberBytes), 0, 64)
			blockReward := fetchBlockReward(blockNumber)

			blockRewardUSD := blockReward * ethPrice

			json_bytes, _ = jsonparser.Set(json_bytes, []byte(strconv.FormatFloat(blockReward, 'f', -1, 64)), "reward")
			json_bytes, _ = jsonparser.Set(json_bytes, []byte(strconv.FormatFloat(blockRewardUSD, 'f', -1, 64)), "reward_usd")

			gasLimitBytes, _, _, _ := jsonparser.Get(json_bytes, "gasLimit")
			gasLimit, _ := strconv.ParseInt(string(gasLimitBytes), 0, 64)
			gasUsedBytes, _, _, _ := jsonparser.Get(json_bytes, "gasUsed")
			gasUsed, _ := strconv.ParseInt(string(gasUsedBytes), 0, 64)
			timestampBytes, _, _, _ := jsonparser.Get(json_bytes, "timestamp")
			timestamp, _ := strconv.ParseInt(string(timestampBytes), 0, 64)

			json_bytes, _ = jsonparser.Set(json_bytes, []byte(strconv.FormatInt(blockNumber, 10)), "number")
			json_bytes, _ = jsonparser.Set(json_bytes, []byte(strconv.FormatInt(gasLimit, 10)), "gasLimit")
			json_bytes, _ = jsonparser.Set(json_bytes, []byte(strconv.FormatInt(gasUsed, 10)), "gasUsed")
			json_bytes, _ = jsonparser.Set(json_bytes, []byte(strconv.FormatInt(timestamp, 10)), "timestamp")

			if ethPrice > 0.0 {
				publisher.Publish(config.REDIS_TOPIC_ETH_BLOCK_HEADER, string(json_bytes))
				if rf != nil {
					rf.Write(string(json_bytes) + "\n")
				}
			}
		}
	}

	// publisher.Close()
	// rf.Close()
}
