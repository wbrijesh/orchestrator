# BASE STAGE: download deps only once
# install git for any go get and download dependencies
FROM golang:1.24.2-alpine AS base
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download


# DEV STAGE: live‑reload with air
# install air and copy code and expose port and start air
FROM base AS dev
RUN apk add --no-cache make
RUN go install github.com/air-verse/air@latest
COPY . .
EXPOSE ${PORT}
ENTRYPOINT ["air", "-c", ".air.toml"]


# BUILD STAGE: compile your binary
# copy code and compile
FROM base AS build
COPY . .
RUN go build -o main cmd/api/main.go


# PROD STAGE: slim, just the binary
# copy compiled binary and run it
FROM alpine:3.20.1 AS prod
WORKDIR /app
COPY --from=build /app/main /app/main
EXPOSE ${PORT}
CMD ["./main"]
