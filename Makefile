PACKAGE_NAME=krossboard-data-processor
DATETIME_VERSION:=$(shell date "+%Y%m%dt%s" | sed 's/\.//g' -)
GIT_SHA:=$(shell git rev-parse --short HEAD)
RELEASE_PACKAGE_NAME=krossboard-v$(DATETIME_VERSION)-$(GIT_SHA)
GOCMD=GO111MODULE=on go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOVENDOR=govendor
GOLANGCI=GO111MODULE=on ./bin/golangci-lint
UPX=upx
PACKER=packer
PACKER_VERSION=1.5.1
PACKER_CONF_FILE="./deploy/packer/cloud-image.json"

all: test build
build:
	$(GOBUILD) -o $(PACKAGE_NAME) -v
build-deps:
	sudo apt-get update && sudo apt-get install -y rrdtool librrd-dev unzip pkg-config upx-ucl unzip
	wget https://releases.hashicorp.com/packer/$(PACKER_VERSION)/packer_$(PACKER_VERSION)_linux_amd64.zip -O /tmp/packer_$(PACKER_VERSION)_linux_amd64.zip
	unzip /tmp/packer_$(PACKER_VERSION)_linux_amd64.zip && sudo mv packer /usr/local/bin/
build-compress: build
	$(UPX) $(PACKAGE_NAME)
docker-build:
	docker run --rm \
		-it -v "$(GOPATH)":/go \
		-w /go/src/bitbucket.org/rsohlich/makepost \
		golang:latest \
		go build -o "$(BINARY_UNIX)" -v
test:
	$(GOCMD) clean -testcache
	$(GOTEST) -v ./...
clean:
	$(GOCLEAN)
	rm -f $(PACKAGE_NAME)
run: build
	./$(PACKAGE_NAME) collector
deps:
	$(GOCMD) get .
tools:
	@if [ ! -f ./bin/golangci-lint ]; then \
		echo "installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.27.0; \
	fi
check: tools
	$(GOLANGCI) run .
dist: build build-compress
	mkdir -p $(RELEASE_PACKAGE_NAME)/scripts/
	cp $(PACKAGE_NAME) $(RELEASE_PACKAGE_NAME)/
	cp scripts/krossboard* $(RELEASE_PACKAGE_NAME)/scripts/
	install -m 755 scripts/install.sh $(RELEASE_PACKAGE_NAME)/
	tar zcf $(RELEASE_PACKAGE_NAME).tgz $(RELEASE_PACKAGE_NAME)
	
dist-cloud-image: dist-cloud-image-aws dist-cloud-image-gcp dist-cloud-image-azure

dist-cloud-image-aws: dist
	$(PACKER) build -only=amazon-ebs \
		-var="release_package_name=$(RELEASE_PACKAGE_NAME)" \
		$(PACKER_CONF_FILE)

dist-cloud-image-gcp: dist
	$(PACKER) build -only=googlecompute \
		-var="release_package_name=$(RELEASE_PACKAGE_NAME)" \
		$(PACKER_CONF_FILE)

dist-cloud-image-azure: dist
	$(PACKER) build -only=azure-arm \
		-var="release_package_name=$(RELEASE_PACKAGE_NAME)" \
		$(PACKER_CONF_FILE)
