# I used chatgpt to help me make this since i apparently have no idea how to use docker
# Stage 1
FROM golang:alpine as builder
RUN apk update && apk add --no-cache git
RUN mkdir /build
ADD . /build/
WORKDIR /build
RUN go get -d -v
RUN go build -o psql-social
# Stage 2
FROM alpine
RUN adduser -S -D -H -h /app appuser
USER appuser
COPY --from=builder /build/ /app/
WORKDIR /app
CMD ["./psql-social"]