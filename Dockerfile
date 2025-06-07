FROM golang:1.24.4 as builder

WORKDIR /project

COPY go.* ./
RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 go build -o bin/server ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates

COPY --from=builder /project/bin/server /bin/server

CMD ["/bin/server"]
