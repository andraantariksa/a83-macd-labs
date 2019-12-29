FROM golang:1.13.5-alpine

WORKDIR /go/src/imgup
COPY . .

EXPOSE 8080

CMD ["./imgup"]