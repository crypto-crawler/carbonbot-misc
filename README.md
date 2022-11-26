# carbonbot-misc

An auxiliary crawler that crawls misc cryptocurrency data.

## 1. Run

```bash
docker run -d --name carbonbot-misc --restart always \
  -e FULL_NODE_URL="wss://mainnet.infura.io/ws/v3/YOUR_PROJECT_ID" \
  -e ETHERSCAN_API_KEY=YOUR_API_KEY \
  -e CMC_API_KEY=YOUR_API_KEY \
  -e REDIS_URL="redis://172.17.0.1:6379" \
  -v "/your/local/path":/carbonbot_data \
  -v "/your/NFS/path":/dest_dir \
  -e DEST_DIR=/dest_dir \
  -u "$(id -u):$(id -g)" soulmachine/carbonbot:misc
```

The `REDIS_URL` environment variable must be present.

## 2. Output Destinations

Crawlers running in the `ghcr.io/crypto-crawler/carbonbot:misc` container write data to the local temporary path `/carbonbot_data` first, then move data to multiple destinations every 15 minutes.

Four kinds of destinations are supported: directory, AWS S3, MinIO and Redis.

### Directory

To save data to a local directory or a NFS directory, users need to mount this directory into the docker container, and specify a `DEST_DIR` environment variable pointing to this directory. For example:

```bash
docker run -d --name carbonbot-trade --restart always -v $YOUR_LOCAL_PATH:/carbonbot_data -v $DEST_DIR:/dest_dir -e DEST_DIR=/dest_dir -u "$(id -u):$(id -g)" ghcr.io/crypto-crawler/carbonbot:misc
```

### AWS S3

To upload data to AWS S3 automatically, uses need to specify three environment variables, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` and `AWS_S3_DIR`. For example:

```bash
docker run -d --name carbonbot-trade --restart always -v $YOUR_LOCAL_PATH:/carbonbot_data -e AWS_ACCESS_KEY_ID="YOUR_ACCESS_KEY" -e AWS_SECRET_ACCESS_KEY="YOUR_SECRET_KEY" -e AWS_S3_DIR="s3://YOUR_BUCKET/path" -u "$(id -u):$(id -g)" ghcr.io/crypto-crawler/carbonbot:misc
```

Optionally, users can specify the `AWS_REGION` environment variable, see [Configuring the AWS SDK for Go
](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html).

### MinIO

To upload data to AWS S3 automatically, users need to specify three environment variables, `MINIO_ACCESS_KEY_ID`, `MINIO_SECRET_ACCESS_KEY`, `MINIO_ENDPOINT_URL` and `MINIO_DIR`. For example:

```bash
docker run -d --name carbonbot-trade --restart always -v $YOUR_LOCAL_PATH:/carbonbot_data -e MINIO_ACCESS_KEY_ID="YOUR_ACCESS_KEY" -e MINIO_SECRET_ACCESS_KEY="YOUR_SECRET_KEY" -e MINIO_ENDPOINT_URL="http://ip:9000" -e MINIO_DIR="minio://YOUR_BUCKET/path" -u "$(id -u):$(id -g)" ghcr.io/crypto-crawler/carbonbot:misc
```

## 3. Build

```bash
docker build -t soulmachine/carbonbot:misc .
docker push soulmachine/carbonbot:misc
```
