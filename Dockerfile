FROM golang:alpine as builder
ARG LDFLAGS=""

RUN apk --update --no-cache add git build-base gcc

COPY . /build
WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

RUN go build -o ./jtso -ldflags "${LDFLAGS}" ./main.go 

FROM alpine:latest

RUN apk update --no-cache 

USER 0
ADD ./html /html
COPY --from=builder /build/jtso /
RUN mkdir -p /etc/jtso
RUN mkdir -p /var/shared/telegraf
RUN mkdir -p /var/shared/grafana

EXPOSE 8081

ENTRYPOINT ["./jtso"]