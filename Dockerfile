FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o taskmanager ./cmd/taskmanager
CMD ["./taskmanager"]
