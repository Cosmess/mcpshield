FROM golang:1.27 AS build

WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -o /out/mcpshield ./cmd/gateway

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/mcpshield /mcpshield
EXPOSE 8080
ENTRYPOINT ["/mcpshield"]