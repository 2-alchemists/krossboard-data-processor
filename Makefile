PRODUCT_NAME=krossboard
PACKAGE_NAME=$(PRODUCT_NAME)-data-processor
PRODUCT_VERSION=$$(grep "ProgramVersion.=.*" main.go | cut -d"\"" -f2)-$$(git rev-parse --short HEAD)
PRODUCT_CLOUD_IMAGE_VERSION=$$(echo $(PRODUCT_VERSION) | sed 's/\.//g' -)
ARCH=$$(uname -m)
DIST_DIR=$(PRODUCT_NAME)-v$(PRODUCT_VERSION)-$(ARCH)
GOCMD=GO111MODULE=off go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get -v
GOVENDOR=govendor
UPX=upx
PACKER=packer
PACKER_VERSION=1.5.1
PACKER_CONF_FILE="./deploy/packer/cloud-image.json"

all: test build
build:
	$(GOBUILD) -o $(PACKAGE_NAME) -v
build-deps:
	sudo apt update && sudo apt install -y rrdtool librrd-dev unzip pkg-config upx-ucl unzip
	wget https://releases.hashicorp.com/packer/$(PACKER_VERSION)/packer_$(PACKER_VERSION)_linux_amd64.zip -O /tmp/packer_$(PACKER_VERSION)_linux_amd64.zip
	unzip /tmp/packer_$(PACKER_VERSION)_linux_amd64.zip && sudo mv packer /usr/local/bin/
build-compress: build
	$(UPX) $(PACKAGE_NAME)
test:
	$(GOCMD) clean -testcache
	$(GOTEST) -v ./...
clean:
	$(GOCLEAN)
	rm -f $(PACKAGE_NAME)
run:
	$(GOBUILD) -o $(PACKAGE_NAME) -v ./...
	./$(PACKAGE_NAME)
deps:
	# cd $GOPATH/src/k8s.io/klog && git checkout v0.4.0 && cd -
	# rm -rf $GOPATH/src/github.com/docker/docker/vendor
	# rm -rf  $GOPATH/src/github.com/docker/distribution/vendor/
	$(GOGET)
vendor:
	$(GOVENDOR) add +external
dist: build build-compress
	mkdir -p $(DIST_DIR)/scripts/
	cp $(PACKAGE_NAME) $(DIST_DIR)/
	cp scripts/$(PRODUCT_NAME)* $(DIST_DIR)/scripts/
	install -m 755 scripts/install.sh $(DIST_DIR)/
	tar zcf $(DIST_DIR).tgz $(DIST_DIR)
	
dist-cloud-image: dist
	$(PACKER) build \
		-var="product_name=$(PRODUCT_NAME)" \
		-var="tarball_version=$(PRODUCT_VERSION)" \
		-var="product_image_version=$(PRODUCT_CLOUD_IMAGE_VERSION)" \
		$(PACKER_CONF_FILE)

dist-cloud-image-aws: dist
	$(PACKER) build -only=amazon-ebs \
		-var="product_name=$(PRODUCT_NAME)" \
		-var="tarball_version=$(PRODUCT_VERSION)" \
		-var="product_image_version=$(PRODUCT_CLOUD_IMAGE_VERSION)" \
		$(PACKER_CONF_FILE)

dist-cloud-image-google: dist
	$(PACKER) build -only=googlecompute \
		-var="product_name=$(PRODUCT_NAME)" \
		-var="tarball_version=$(PRODUCT_VERSION)" \
		-var="product_image_version=$(PRODUCT_CLOUD_IMAGE_VERSION)" \
		$(PACKER_CONF_FILE)

dist-cloud-image-azure: dist
	$(PACKER) build -only=azure-arm \
		-var="product_name=$(PRODUCT_NAME)" \
		-var="tarball_version=$(PRODUCT_VERSION)" \
		-var="product_image_version=$(PRODUCT_CLOUD_IMAGE_VERSION)" \
		$(PACKER_CONF_FILE)

# Cross compilation
docker-build:
	docker run --rm -it -v "$(GOPATH)":/go -w /go/src/bitbucket.org/rsohlich/makepost golang:latest go build -o "$(BINARY_UNIX)" -v