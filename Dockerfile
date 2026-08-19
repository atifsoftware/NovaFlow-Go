FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /novaflow .

FROM alpine:3.20
WORKDIR /app
COPY --from=build /novaflow /app/novaflow
COPY app/views ./app/views
COPY public ./public
COPY database ./database
COPY .env.example ./.env
EXPOSE 8080
CMD ["/app/novaflow"]
