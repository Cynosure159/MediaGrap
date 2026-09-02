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

# The production runtime contains a purpose-built static ffprobe. The build
# enables only the container/protocol/parser pieces needed by MediaGrap, so
# the final image does not include the full Alpine ffmpeg dependency graph.
# `ffprobe` is the default (and only) production runtime target.
FROM alpine:3.22 AS ffprobe-builder
ARG FFMPEG_VERSION=7.1.1
RUN apk add --no-cache build-base ca-certificates curl nasm xz
WORKDIR /src
RUN curl -fsSL "https://ffmpeg.org/releases/ffmpeg-${FFMPEG_VERSION}.tar.xz" \
    | tar -xJ --strip-components=1
RUN ./configure \
      --prefix=/usr \
      --bindir=/usr/bin \
      --disable-autodetect \
      --disable-debug \
      --disable-doc \
      --disable-everything \
      --disable-avdevice \
      --disable-avfilter \
      --disable-ffmpeg \
      --disable-ffplay \
      --disable-network \
      --disable-swresample \
      --disable-swscale \
      --enable-ffprobe \
      --enable-protocol=file \
      --enable-demuxer=avi,flac,matroska,mp3,mpegts,mov,ogg,wav \
      --enable-parser=aac,ac3,av1,h264,hevc,mpegaudio,mpeg4video,opus,vorbis,vp9 \
      --enable-small \
      --enable-static \
      --disable-shared \
      --extra-cflags="-Os" \
      --extra-ldflags="-static"
RUN make -j2 && make install DESTDIR=/out

FROM scratch AS ffprobe
COPY --from=ffprobe-builder /out/usr/bin/ffprobe /usr/bin/ffprobe
COPY --from=ffprobe-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --chown=65532:65532 --from=go-builder /out/mediagrap /mediagrap
ENV MEDIAGRAP_FFPROBE_PATH=/usr/bin/ffprobe
USER 65532:65532
EXPOSE 8080
VOLUME ["/config", "/cache"]
ENTRYPOINT ["/mediagrap"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD ["/mediagrap", "healthcheck"]
