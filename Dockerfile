FROM golang:1.23.12-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/valdrd ./cmd/valdrd && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/valdr-cli ./cmd/valdr-cli && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/valdr-miner ./cmd/valdr-miner && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/valdr-explorer ./cmd/valdr-explorer

FROM debian:bookworm-slim AS runtime

RUN groupadd --system --gid 10001 valdr && \
    useradd --system --uid 10001 --gid 10001 --home-dir /var/lib/valdr --shell /usr/sbin/nologin valdr && \
    mkdir -p /var/lib/valdr /etc/valdr && \
    chown -R valdr:valdr /var/lib/valdr /etc/valdr

COPY --from=build /out/valdrd /usr/local/bin/valdrd
COPY --from=build /out/valdr-cli /usr/local/bin/valdr-cli
COPY --from=build /out/valdr-miner /usr/local/bin/valdr-miner
COPY --from=build /out/valdr-explorer /usr/local/bin/valdr-explorer

USER valdr:valdr
WORKDIR /var/lib/valdr
VOLUME ["/var/lib/valdr"]
EXPOSE 17333 8080

ENTRYPOINT ["valdrd"]
CMD ["version"]
