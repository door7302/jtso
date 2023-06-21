FROM golang:alpine as builder
ARG LDFLAGS=""

RUN apk --update --no-cache add git build-base gcc

COPY . /build
WORKDIR /build

RUN go build -o ./jtso -ldflags "${LDFLAGS}" ./main.go 

FROM alpine:latest

RUN apk update --no-cache && \
    adduser -S -D -H -h / jtso

USER jtso
COPY --from=builder /build/jtso /

RUN mkdir -p /etc/jtso

EXPOSE 8081

ENTRYPOINT ["./jtso --config /etc/jtso/config.yml"]