FROM golang:1.18.0-buster as builder

COPY src /go/src/app
WORKDIR /go/src/app
RUN go get ./...

# build the source
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main main.go

FROM alpine:3.7
COPY sql/ /sql/
WORKDIR /root

ENV BIND_ADDRESS "0.0.0.0:8080"
ENV TOKEN_HEADER_IN "Authorization"
ENV DATABASE_CONNECTION_STRING "postgres://user:password@localhost:5432/postgres"
ENV JWKS_URL ""
ENV MIGRATION_FILES="/sql/schema"

# copy the binary from builder
COPY --from=builder /go/src/app/main .
# run the binary
CMD ["./main"]