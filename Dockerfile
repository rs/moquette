# build stage
FROM golang:1.11-rc-stretch AS build-env
WORKDIR /src
COPY . /src
RUN CGO_ENABLED=0 go build -o moquette

# final stage
FROM alpine
RUN apk add --no-cache bash
COPY --from=build-env /src/moquette /app/moquette
ENTRYPOINT ["/app/moquette"]
