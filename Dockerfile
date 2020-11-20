FROM golang:1.15.0-buster as builder

RUN go get github.com/dgrijalva/jwt-go
RUN go get github.com/DanielHons/go-jwt-exchange/jwt_exchange
WORKDIR /go/src/app
ADD main.go main.go
# build the source
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main main.go

COPY sql/ /sql/
FROM alpine:3.7
WORKDIR /root

ENV BIND_ADDRESS "0.0.0.0:8080"
ENV TOKEN_HEADER_IN "Authorization"
ENV DATABASE_CONNECTION_STRING "postgres://user:password@localhost:5432/postgres"
ENV JWKS_URL ""
ENV MIGRATION_FILES="/sql"

# copy the binary from builder
COPY --from=builder /go/src/app/main .
# run the binary
CMD ["./main"]