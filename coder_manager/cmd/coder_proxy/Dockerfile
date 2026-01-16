FROM golang:1.22 AS build

WORKDIR /app

COPY . .

WORKDIR /app/coder_manager
RUN go mod download
RUN CGO_ENABLED=0 go build -o /out/coder_manager_proxy ./cmd/coder_proxy

FROM gcr.io/distroless/base-debian12

COPY --from=build /out/coder_manager_proxy /coder_manager_proxy

ENTRYPOINT ["/coder_manager_proxy"]
