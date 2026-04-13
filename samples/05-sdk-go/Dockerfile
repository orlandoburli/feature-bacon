FROM golang:1.23-alpine AS builder
WORKDIR /build
COPY sdks/go/ /build/sdks/go/
COPY samples/05-sdk-go/ /build/app/
WORKDIR /build/app
RUN go build -o /product-service .

FROM alpine:3.20
RUN adduser -D appuser
COPY --from=builder /product-service /product-service
USER appuser
EXPOSE 3000
CMD ["/product-service"]
