-include .env.local
export MEDIAGRAP_MEDIA_ROOTS

.PHONY: dev api web build test test-go test-web typecheck format docker-build prepare-ui

dev:
	@$(MAKE) -j2 api web

api:
	MEDIAGRAP_CONFIG_DIR=$${MEDIAGRAP_CONFIG_DIR:-./.local/config} MEDIAGRAP_CACHE_DIR=$${MEDIAGRAP_CACHE_DIR:-./.local/cache} go run ./cmd/mediagrap --log-format=text

web:
	npm --prefix web run dev

prepare-ui:
	npm --prefix web run build
	rm -rf internal/httpapi/ui/dist/assets internal/httpapi/ui/dist/index.html internal/httpapi/ui/dist/manifest.webmanifest internal/httpapi/ui/dist/sw.js internal/httpapi/ui/dist/workbox-*.js
	mkdir -p internal/httpapi/ui/dist
	cp -R web/dist/. internal/httpapi/ui/dist/

build: prepare-ui
	go build -trimpath -ldflags "-s -w" -o bin/mediagrap ./cmd/mediagrap

test: test-go test-web

test-go:
	go test -race ./...

test-web:
	npm --prefix web run test

typecheck:
	npm --prefix web run typecheck

format:
	gofmt -w cmd internal

docker-build:
	docker build -t mediagrap:dev .
