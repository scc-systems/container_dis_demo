# ---- build ----
FROM golang:1.23-alpine AS build
ENV GOTOOLCHAIN=local
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN mkdir -p /out && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /out/wallsync ./cmd/wallsync

# ---- runtime ----
FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/wallsync /wallsync
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/wallsync"]
