FROM golang:1.25.8-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./

RUN go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=dev
ARG REVISION=unknown
ARG BUILT_AT=unknown
ARG GOPROXY=https://goproxy.cn,direct
ARG GOSUMDB=sum.golang.google.cn

ENV GOPROXY=${GOPROXY}
ENV GOSUMDB=${GOSUMDB}

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath \
    -ldflags="-s -w -X main.version=${VERSION} -X main.revision=${REVISION} -X main.builtAt=${BUILT_AT}" \
    -o /out/techstack .

FROM alpine:3.22

ENV TZ=Asia/Shanghai
ENV TECHSTACK_CONFIG=/etc/techstack/config.yml

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S techstack \
    && adduser -S -G techstack -h /var/lib/techstack techstack \
    && mkdir -p /opt/techstack /etc/techstack /var/lib/techstack/cache \
    && chown -R techstack:techstack /opt/techstack /etc/techstack /var/lib/techstack

COPY --from=builder /out/techstack /opt/techstack/techstack
COPY --chown=techstack:techstack config.yml.tpl /etc/techstack/config.yml.tpl

WORKDIR /opt/techstack

USER techstack:techstack

EXPOSE 8775

VOLUME ["/var/lib/techstack"]

CMD ["/opt/techstack/techstack", "daemon"]
