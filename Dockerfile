FROM golang:1.21-alpine

WORKDIR /app

#COPY go.mod ./
#RUN go mod download

COPY . .
#RUN cat /app/node/node.go
RUN go build -o /node ./cmd/node
#RUN chmod +x ./node
#RUN apk add openvpn

EXPOSE 5555/udp
CMD [ "/node", "-port", "5555" ]