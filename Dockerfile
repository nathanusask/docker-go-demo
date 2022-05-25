FROM golang:1.18.2

WORKDIR /app

COPY . .

RUN go build -o /demo
RUN rm -rf ./*

CMD ["/demo"]