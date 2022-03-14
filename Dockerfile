FROM golang:1.17.8-alpine3.15 as builder
#RUN apk update
RUN apk add git
WORKDIR /
COPY go.mod .
COPY go.sum .
COPY main.go .
# Disable defualt GOPROXY
RUN go env -w GOPROXY=direct
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o main.bin .

FROM scratch
#FROM alpine:3.15
WORKDIR /
COPY --from=builder main.bin main
EXPOSE 8080
ENTRYPOINT ["/main"]
