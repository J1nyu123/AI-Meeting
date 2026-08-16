FROM golang:1.24-alpine AS build
RUN apk add --no-cache poppler-utils
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
ARG TARGET=api
RUN CGO_ENABLED=0 go build -trimpath -o /out/app ./cmd/${TARGET}

FROM alpine:3.21
RUN apk add --no-cache ca-certificates poppler-utils tzdata \
    && adduser -D -u 10001 app \
    && mkdir -p /app/data/uploads \
    && chown -R app:app /app/data
USER app
WORKDIR /app
COPY --from=build /out/app /app/app
COPY migrations /app/migrations
EXPOSE 8080
ENTRYPOINT ["/app/app"]
