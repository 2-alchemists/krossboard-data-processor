PACKAGE_NAME=koamc-cluster-manager
VERSION=$$(grep "ProgramVersion.=.*" main.go | cut -d"\"" -f2)
ARCH=$$(uname -m)
DIST_DIR=$(PACKAGE_NAME)-v$(VERSION)-$(ARCH)
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get -v
GOVENDOR=govendor
UPX=upx
PACKER=packer
PACKER_VAR_FILE="./deploy/packer/variables.json"
PACKER_CONF_FILE="./deploy/packer/cloud-image.json"

all: test build
build:
	$(GOBUILD) -o $(PACKAGE_NAME) -v
compress: build
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
	$(GOGET)
vendor:
	$(GOVENDOR) add +external
dist: build compress
	mkdir -p $(DIST_DIR)/scripts/
	cp $(PACKAGE_NAME) $(DIST_DIR)/
	cp scripts/$(PACKAGE_NAME)* $(DIST_DIR)/scripts
	install -m 755 scripts/install.sh $(DIST_DIR)/
	tar zcf $(DIST_DIR).tgz $(DIST_DIR)
cloud-image:
	$(PACKER) build -var-file=$(PACKER_VAR_FILE) $(PACKER_CONF_FILE)	
dist-cloud-image: dist
	$(PACKER) build -var-file=$(PACKER_VAR_FILE) $(PACKER_CONF_FILE)

# Cross compilation
docker-build:
	docker run --rm -it -v "$(GOPATH)":/go -w /go/src/bitbucket.org/rsohlich/makepost golang:latest go build -o "$(BINARY_UNIX)" -v