FROM golang:1.21-alpine

WORKDIR /app

#COPY go.mod ./
#RUN go mod download

COPY . .
#RUN cat /app/node/node.go
RUN go build -o /node ./cmd/node
#RUN chmod +x ./node
#RUN apk add openvpn

EXPOSE 12000
CMD [ "/node", "-id", "1000", "-port", "12000" ]