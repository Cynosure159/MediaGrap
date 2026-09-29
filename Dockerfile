# Release builds consume only scripts/release-context.py output, not a live checkout.
FROM node:24-alpine AS web-builder
WORKDIR /workspace/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM alpine:3.22 AS ffprobe-builder
RUN apk add --no-cache build-base ca-certificates ca-certificates-bundle curl nasm xz
WORKDIR /materials
RUN curl -fsSLo ffmpeg-7.1.1.tar.xz https://ffmpeg.org/releases/ffmpeg-7.1.1.tar.xz \
 && echo '733984395e0dbbe5c046abda2dc49a5544e7e0e1e2366bba849222ae9e3a03b1  ffmpeg-7.1.1.tar.xz' | sha256sum -c - \
 && curl -fsSLo musl-1.2.5.tar.gz https://musl.libc.org/releases/musl-1.2.5.tar.gz \
 && echo 'a9a118bbe84d8764da0ea0d28b3ab3fae8477fc7e4085d90102b8596fc7c75e4  musl-1.2.5.tar.gz' | sha256sum -c - \
 && curl -fsSLo ca-certificates-20260611.tar.bz2 https://distfiles.alpinelinux.org/distfiles/v3.22/ca-certificates-20260611.tar.bz2 \
 && echo '32ca73f2e81e2b88dc614f12e1ee04a82b1ec5a8e29d9f359ddf8905a0afcbb0  ca-certificates-20260611.tar.bz2' | sha256sum -c -
# Refuse silent upgrades whose source/notice inventory has not been reviewed.
RUN apk list --installed musl | grep -q '^musl-1.2.5-r12 ' \
 && test "$(gcc -dumpfullversion)" = '14.2.0'
WORKDIR /src
RUN tar -xJf /materials/ffmpeg-7.1.1.tar.xz --strip-components=1
COPY scripts/build-ffprobe.sh /build-ffprobe.sh
RUN sh /build-ffprobe.sh
COPY docs/legal/ /materials/notices/
RUN tar -xOf /materials/musl-1.2.5.tar.gz musl-1.2.5/COPYRIGHT > /materials/notices/musl-COPYRIGHT

FROM golang:1.26-alpine AS go-builder
ARG TARGETARCH
RUN apk add --no-cache python3
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
COPY --from=web-builder /workspace/web/ ./web/
COPY --from=web-builder /workspace/web/dist/ ./internal/httpapi/ui/dist/
COPY --from=ffprobe-builder /materials/ /materials/
# Versions come from the reviewed input manifest, not independently overridable args.
RUN export VERSION="$(python3 -c 'import json; print(json.load(open("release-input.json"))["version"])')" \
 && export COMMIT="$(python3 -c 'import json; print(json.load(open("release-input.json"))["commit"])')" \
 && export BUILT_AT="$(python3 -c 'import json; print(json.load(open("release-input.json"))["builtAt"])')" \
 && python3 scripts/package-source.py \
 && export SOURCE_URL="$(python3 -c 'import json; print(json.load(open("/out/source/manifest.json"))["url"])')" \
 && export SOURCE_SHA="$(python3 -c 'import json; print(json.load(open("/out/source/manifest.json"))["sha256"])')" \
 && CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.builtAt=${BUILT_AT} -X github.com/mediagrap/mediagrap/internal/httpapi.sourceURL=${SOURCE_URL} -X github.com/mediagrap/mediagrap/internal/httpapi.sourceSHA256=${SOURCE_SHA} -X github.com/mediagrap/mediagrap/internal/httpapi.sourceArchitecture=${TARGETARCH}" -o /out/mediagrap ./cmd/mediagrap

FROM scratch AS artifacts
COPY --from=go-builder /out/mediagrap /mediagrap
COPY --from=go-builder /out/source/ /distribution/
COPY --from=ffprobe-builder /materials/ /materials/
COPY --from=ffprobe-builder /out/usr/bin/ffprobe /ffprobe

FROM scratch AS ffprobe
COPY --from=ffprobe-builder /out/usr/bin/ffprobe /usr/bin/ffprobe
COPY --from=ffprobe-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=ffprobe-builder /materials/notices/ /usr/share/mediagrap/notices/
COPY --from=ffprobe-builder /materials/ffmpeg/ /usr/share/mediagrap/ffmpeg/
COPY --from=ffprobe-builder /materials/runtime-sha256.txt /usr/share/mediagrap/runtime-sha256.txt
COPY LICENSE /usr/share/mediagrap/LICENSE
COPY --chown=65532:65532 --from=go-builder /out/mediagrap /mediagrap
ENV MEDIAGRAP_FFPROBE_PATH=/usr/bin/ffprobe
USER 65532:65532
EXPOSE 8080
VOLUME ["/config", "/cache"]
ENTRYPOINT ["/mediagrap"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD ["/mediagrap", "healthcheck"]
