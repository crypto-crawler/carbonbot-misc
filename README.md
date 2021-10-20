# carbonbot-misc

An auxiliary crawler that crawls misc cryptocurrency data.

## Run

```bash
docker run -d --name carbonbot-misc --restart always -v "/your/local/path":/data -e AWS_ACCESS_KEY_ID=YOUR_ACCESS_KEY -e AWS_SECRET_ACCESS_KEY=YOUR_SECRET_KEY -e AWS_S3_DIR="s3://to/your/path" -e REDIS_URL="redis://:password@ip:6379" -e FULL_NODE_URL="wss://mainnet.infura.io/ws/v3/YOUR_PROJECT_ID" -e ETHERSCAN_API_KEY=YOUR_API_KEY -u "$(id -u):$(id -g)" soulmachine/carbonbot:misc
```

## Build

```bash
docker pull golang:latest && docker pull node:bullseye-slim
docker build -t soulmachine/carbonbot:misc .
docker push soulmachine/carbonbot:misc
```
