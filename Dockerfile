FROM golang:latest

WORKDIR /app

COPY . .

# Remove unused dependencies and update go.mod
RUN go mod tidy

RUN go build -o ./bin/web ./cmd/web 
VOLUME ["/app"]
CMD ["/app/bin/web"]

EXPOSE 8080