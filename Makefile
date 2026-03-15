IMAGE ?= contrib-sync:dev
GO_IMAGE ?= golang:1.23-alpine
RUNTIME_IMAGE ?= alpine:3.20

.PHONY: build run test shell clean

build:
	docker build \
		--build-arg GO_IMAGE=$(GO_IMAGE) \
		--build-arg RUNTIME_IMAGE=$(RUNTIME_IMAGE) \
		-t $(IMAGE) .

run:
	docker run --rm \
		-v $(PWD)/config.yaml:/config.yaml:ro \
		-v $(HOME)/.contrib-mirror:/mirror \
		$(IMAGE) sync --config /config.yaml

test:
	docker run --rm \
		-v $(PWD):/src \
		-w /src \
		$(GO_IMAGE) \
		sh -lc 'export PATH=/usr/local/go/bin:$$PATH && go test ./...'

shell:
	docker run --rm -it \
		-v $(PWD):/src \
		-w /src \
		$(GO_IMAGE) sh

clean:
	docker rmi $(IMAGE) 2>/dev/null || true
