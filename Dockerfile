# Builder
FROM whatwewant/builder-go:v1.19-1 as builder

WORKDIR /build

COPY go.mod ./

COPY go.sum ./

RUN go mod download

COPY . .

RUN GOOS=linux \
  GOARCH=amd64 \
  go build \
  -trimpath \
  -ldflags '-w -s -buildid=' \
  -v -o chatgpt-for-feishu-bot

# Server
FROM whatwewant/go:v1.19-1

LABEL MAINTAINER="Zero<tobewhatwewant@gmail.com>"

LABEL org.opencontainers.image.source="https://github.com/go-zoox/chatgpt-for-feishu-bot"

ARG VERSION=latest

ENV MODE=production

COPY --from=builder /build/chatgpt-for-feishu-bot /bin

ENV VERSION=${VERSION}

CMD chatgpt-for-feishu-bot server -c /conf/config.yml
