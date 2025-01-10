# stage one
FROM registry.verystar.net/library/golang:1.23 AS builder
WORKDIR /build
ENV GOPROXY=https://goproxy.verystar.net,https://goproxy.cn,direct \
    GOPRIVATE="" \
    GONOSUMDB="git.verystar.cn" \
    GOSUMDB="sum.golang.google.cn"
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app ./cmd/tkn


# stage two
FROM registry.verystar.net/library/debian:12
WORKDIR /app
COPY --from=builder /build/app /usr/local/bin/tkn
ENTRYPOINT [ "/usr/local/bin/tkn" ]
CMD [ "--help" ]