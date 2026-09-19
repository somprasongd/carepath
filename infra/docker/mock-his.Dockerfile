FROM golang:1.25-alpine AS build
WORKDIR /src
COPY apps/mock-his/go.mod ./
RUN go mod download
COPY apps/mock-his .
RUN CGO_ENABLED=0 go build -o /out/mock-his ./cmd/server

FROM alpine:3.22
COPY --from=build /out/mock-his /mock-his
EXPOSE 8090
ENTRYPOINT ["/mock-his"]
