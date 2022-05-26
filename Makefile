PACKAGE_NAME=krossboard-data-processor
PROGRAM_ARTIFACT=./bin/krossboard-data-processor
DATETIME_VERSION:=$(shell date "+%Y%m%dt%s" | sed 's/\.//g' -)
GIT_SHA:=$(shell git rev-parse --short HEAD)
RELEASE_PUBLIC_VERSION="v$(shell ./tooling/get-dist-version.sh)"
RELEASE_PACKAGE_PUBLIC=krossboard-$(RELEASE_PUBLIC_VERSION)
RELEASE_PACKAGE_CLOUD=krossboard-v$(DATETIME_VERSION)-$(GIT_SHA)
GOCMD=GO111MODULE=on go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOVENDOR=govendor
GOIMAGE=golang:1.18.2
GOLANGCI=GO111MODULE=on ./bin/golangci-lint
UPX=upx
PACKER=packer
PACKER_VERSION=1.6.2
PACKER_CONF_FILE="./deploy/packer/cloud-image.json"

all: test build

dist-check-prereqs:
	test -n "$(KROSSBOARD_KOAINSTANCE_IMAGE)"
	test -n "$(KROSSBOARD_UI_IMAGE)"

build-deps:
	sudo apt-get update && sudo apt-get install -y rrdtool librrd-dev unzip pkg-config upx-ucl unzip
	wget https://releases.hashicorp.com/packer/$(PACKER_VERSION)/packer_$(PACKER_VERSION)_linux_amd64.zip -O /tmp/packer_$(PACKER_VERSION)_linux_amd64.zip
	unzip /tmp/packer_$(PACKER_VERSION)_linux_amd64.zip && sudo mv packer /usr/local/bin/

build:
	$(GOBUILD) -o $(PROGRAM_ARTIFACT) -v

build-compress: build
	$(UPX) $(PROGRAM_ARTIFACT)

build-docker:
	docker run --rm \
		-it \
		-v "$(PWD)":/go \
		-e GOOS=linux \
		-e GOARCH=amd64 \
		-w /go \
		golang:$(GOVERSION) \
		go build -o $(PROGRAM_ARTIFACT) .

build-ci:
	docker run --rm \
	-e GO111MODULE=on \
	-e CGO_ENABLED=0 \
	-e GOOS=linux \
	-e GOARCH=amd64 \
	-v "$(PWD)":/app \
	-w /app \
	$(GOIMAGE) \
	go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o "./bin/$(PACKAGE_NAME)" -v

test:
	$(GOCMD) clean -testcache
	$(GOTEST) -v ./...

clean:
	$(GOCLEAN)
	rm -f $(PACKAGE_NAME)

run: build
	./$(PROGRAM_ARTIFACT) collector

deps:
	$(GOCMD) mod tidy

tools:
	@if [ ! -f ./bin/golangci-lint ]; then \
		echo "installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.32.2; \
	fi

check: tools
	$(GOLANGCI) run .

dist-cloud: dist-check-prereqs build build-compress
	./tooling/create-distrib-package.sh $(PROGRAM_ARTIFACT) $(RELEASE_PACKAGE_CLOUD) $(KROSSBOARD_KOAINSTANCE_IMAGE) $(KROSSBOARD_UI_IMAGE)

dist-public: dist-check-prereqs build build-compress
	./tooling/create-distrib-package.sh $(PROGRAM_ARTIFACT) $(RELEASE_PACKAGE_PUBLIC) $(KROSSBOARD_KOAINSTANCE_IMAGE) $(KROSSBOARD_UI_IMAGE)

dist-cloud-image-aws: dist-cloud
	$(PACKER) build -only=amazon-ebs \
		-var="release_package_name=$(RELEASE_PACKAGE_CLOUD)" \
		$(PACKER_CONF_FILE)

dist-cloud-image-gcp: dist-cloud
	$(PACKER) build -only=googlecompute \
		-var="release_package_name=$(RELEASE_PACKAGE_CLOUD)" \
		$(PACKER_CONF_FILE)

dist-cloud-image-azure: dist-cloud
	$(PACKER) build -only=azure-arm \
		-var="release_package_name=$(RELEASE_PACKAGE_CLOUD)" \
		$(PACKER_CONF_FILE)

dist-ovf: dist-public
	$(PACKER) build -only=virtualbox-iso \
		-var="release_package_name=$(RELEASE_PACKAGE_PUBLIC)" \
		$(PACKER_CONF_FILE)

publish-ovf: dist-ovf
	./tooling/publish-release.sh $(RELEASE_PUBLIC_VERSION) $(RELEASE_PACKAGE_PUBLIC)





