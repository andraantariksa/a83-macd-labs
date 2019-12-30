FROM golang:1.13.5-alpine

WORKDIR /go/src/imgup
COPY . .

RUN go mod download
RUN go build -o imgup

EXPOSE 8080

CMD /go/src/imgup/imgup