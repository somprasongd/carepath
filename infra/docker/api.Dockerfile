FROM golang:1.25-alpine AS build
WORKDIR /src
COPY apps/api/go.mod ./
RUN go mod download
COPY apps/api .
RUN CGO_ENABLED=0 go build -o /out/carepath-api ./cmd/server

FROM alpine:3.22
COPY --from=build /out/carepath-api /carepath-api
EXPOSE 8080
ENTRYPOINT ["/carepath-api"]
