FROM golang:1.21 as builder
RUN mkdir /app
ADD . /app
WORKDIR /app

EXPOSE 10000

RUN CGO_ENABLED=0 GOOS=linux go build -o crashdummy main.go

RUN chmod +x ./crashdummy

FROM alpine:latest as production
COPY --from=builder /app .

RUN chmod +x /crashdummy

CMD ["/crashdummy"]