FROM node:24-alpine AS web-builder
WORKDIR /workspace/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26-alpine AS go-builder
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
COPY --from=web-builder /workspace/web/dist/ ./internal/httpapi/ui/dist/
ARG VERSION=dev
ARG COMMIT=none
ARG BUILT_AT=unknown
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.builtAt=${BUILT_AT}" -o /out/mediagrap ./cmd/mediagrap

FROM gcr.io/distroless/static-debian12:nonroot
COPY --chown=nonroot:nonroot --from=go-builder /out/mediagrap /mediagrap
USER nonroot:nonroot
EXPOSE 8080
VOLUME ["/config", "/cache"]
ENTRYPOINT ["/mediagrap"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD ["/mediagrap", "healthcheck"]
