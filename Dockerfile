FROM golang:latest AS go_builder

RUN mkdir /project
WORKDIR /project
COPY ./ ./
RUN go build -o cmc_global_metrics cmd/cmc_global_metrics/main.go \
 && go build -o cmc_price_crawler cmd/cmc_price_crawler/main.go \
 && go build -o crawler_block_header cmd/crawler_block_header/main.go \
 && go build -o crawler_gas_price cmd/crawler_gas_price/main.go \
 && go build -o mark_price cmd/mark_price/main.go \
 && go build -o ftx_spot_price cmd/ftx_spot_price/main.go


FROM node:bullseye-slim

COPY --from=go_builder /project/cmc_global_metrics /usr/local/bin/
COPY --from=go_builder /project/cmc_price_crawler /usr/local/bin/
COPY --from=go_builder /project/crawler_block_header /usr/local/bin/
COPY --from=go_builder /project/crawler_gas_price /usr/local/bin/
COPY --from=go_builder /project/mark_price /usr/local/bin/
COPY --from=go_builder /project/ftx_spot_price /usr/local/bin/

# procps provides the ps command, which is needed by pm2
RUN apt-get -qy update && apt-get -qy --no-install-recommends install \
    ca-certificates curl procps pigz \
 && npm install pm2 -g --production \
 && apt-get -qy install gzip unzip && curl https://rclone.org/install.sh | bash \
 && apt-get -qy autoremove && apt-get clean && rm -rf /var/lib/apt/lists/* && rm -rf /tmp/*

# Install fixuid
RUN ARCH="$(dpkg --print-architecture)" && \
    curl -SsL https://github.com/boxboat/fixuid/releases/download/v0.5.1/fixuid-0.5.1-linux-amd64.tar.gz | tar -C /usr/local/bin -xzf - && \
    chown root:root /usr/local/bin/fixuid && \
    chmod 4755 /usr/local/bin/fixuid && \
    mkdir -p /etc/fixuid && \
    printf "user: node\ngroup: node\n" > /etc/fixuid/config.yml

COPY --chown=node:node ./conf/rclone.conf /home/node/.config/rclone/rclone.conf
COPY --chown=node:node ./conf/pm2.misc.config.js /home/node/pm2.misc.config.js
COPY ./conf/upload.sh /usr/local/bin/upload.sh

ENV RUST_LOG "warn"
ENV RUST_BACKTRACE 1

VOLUME [ "/carbonbot_data" ]
ENV DATA_DIR /carbonbot_data

USER node:node
ENV USER node
WORKDIR /home/node

ENTRYPOINT ["fixuid", "-q"]

CMD [ "pm2-runtime", "start", "pm2.misc.config.js" ]
