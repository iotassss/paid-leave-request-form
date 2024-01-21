FROM golang:1.21.6
WORKDIR /app
RUN go install github.com/cosmtrek/air@latest
CMD ["air"]
