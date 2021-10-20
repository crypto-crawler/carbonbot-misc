package main

import (
	"context"
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

func fetch_cmc_global_metrics(rf *utils.RollingFile) string {
	url := "https://pro-api.coinmarketcap.com/v1/global-metrics/quotes/latest"
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)

	cmc_api_key := os.Getenv("CMC_API_KEY")
	if len(cmc_api_key) == 0 {
		log.Fatal("The CMC_API_KEY environment variable is empty")
	}
	req.Header.Set("X-CMC_PRO_API_KEY", cmc_api_key)
	resp, err := client.Do(req)
	if err != nil {
		panic(err.Error())
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	if rf != nil {
		rf.Write(string(body) + "\n")
	}
	data, _, _, _ := jsonparser.Get(body, "data")

	usd, _, _, _ := jsonparser.Get(data, "quote", "USD")
	data = jsonparser.Delete(data, "quote")
	last_updated, _, _, _ := jsonparser.Get(data, "last_updated")
	data = jsonparser.Delete(data, "last_updated")

	jsonparser.ObjectEach(usd, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		data, _ = jsonparser.Set(data, value, string(key))
		return nil
	})

	last_updated = []byte("\"" + string(last_updated) + "\"")
	data, _ = jsonparser.Set(data, last_updated, "last_updated") // TODO: workaround jsonparser/issues/218

	return string(data)
}

func main() {
	ctx := context.Background()

	data_dir := os.Getenv("DATA_DIR")
	var rf *utils.RollingFile
	if len(data_dir) == 0 {
		log.Println("The DATA_DIR environment variable is empty")
		rf = nil
	} else {
		rf = utils.NewRollingFile(data_dir, "cmc.global_metrics")
	}

	redis_url := os.Getenv("REDIS_URL")
	if len(redis_url) == 0 {
		log.Println("The REDIS_URL environment variable is empty")
	} else {
		utils.WaitRedis(ctx, redis_url)
	}

	publisher := pubsub.NewPublisher(ctx, redis_url)

	ticker := time.NewTicker(10 * time.Minute) // crawl every 10 minutes
	defer ticker.Stop()

	for range ticker.C {
		metrics := fetch_cmc_global_metrics(rf)
		publisher.Publish(config.REDIS_TOPIC_CMC_GLOBAL_METRICS, string(metrics))
	}

	rf.Close()
	publisher.Close()
}
