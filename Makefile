PACKAGE_NAME=krossboard-data-processor
PACKAGE_BUILD_ARTIFACT=./bin/krossboard-data-processor
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
PACKER_VERSION=1.6.2
PACKER_CONF_FILE="./deploy/packer/cloud-image.json"

all: test build

build:
	$(GOBUILD) -o $(PACKAGE_BUILD_ARTIFACT) -v

build-deps:
	sudo apt-get update && sudo apt-get install -y rrdtool librrd-dev unzip pkg-config upx-ucl unzip
	wget https://releases.hashicorp.com/packer/$(PACKER_VERSION)/packer_$(PACKER_VERSION)_linux_amd64.zip -O /tmp/packer_$(PACKER_VERSION)_linux_amd64.zip
	unzip /tmp/packer_$(PACKER_VERSION)_linux_amd64.zip && sudo mv packer /usr/local/bin/

build-compress: build
	$(UPX) $(PACKAGE_BUILD_ARTIFACT)

docker-build:
	docker run --rm \
		-it -v "$(GOPATH)":/go \
		-w /go/src/bitbucket.org/rsohlich/makepost \
		golang:latest \
		go build -o $(PACKAGE_BUILD_ARTIFACT) -v
test:
	$(GOCMD) clean -testcache
	$(GOTEST) -v ./...

clean:
	$(GOCLEAN)
	rm -f $(PACKAGE_NAME)

run: build
	./$(PACKAGE_BUILD_ARTIFACT) collector

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
	cp $(PACKAGE_BUILD_ARTIFACT) $(RELEASE_PACKAGE_NAME)/
	cp ./scripts/krossboard* $(RELEASE_PACKAGE_NAME)/scripts/
	install -m 755 ./scripts/install.sh $(RELEASE_PACKAGE_NAME)/
	tar zcf $(RELEASE_PACKAGE_NAME).tgz $(RELEASE_PACKAGE_NAME)
	rm -rf $(RELEASE_PACKAGE_NAME)/

check-cloud-image-pre:
	test -n "$(KROSSBOARD_KOAINSTANCE_IMAGE)"
	test -n "$(KROSSBOARD_UI_IMAGE)"

dist-cloud-image-aws: check-cloud-image-pre dist
	$(PACKER) build -only=amazon-ebs \
		-var="release_package_name=$(RELEASE_PACKAGE_NAME)" \
		$(PACKER_CONF_FILE)

dist-cloud-image-gcp: check-cloud-image-pre dist
	$(PACKER) build -only=googlecompute \
		-var="release_package_name=$(RELEASE_PACKAGE_NAME)" \
		$(PACKER_CONF_FILE)

dist-cloud-image-azure: check-cloud-image-pre dist
	$(PACKER) build -only=azure-arm \
		-var="release_package_name=$(RELEASE_PACKAGE_NAME)" \
		$(PACKER_CONF_FILE)

dist-ovf-image: check-cloud-image-pre dist
	$(PACKER) build -only=virtualbox-iso \
		-var="release_package_name=$(RELEASE_PACKAGE_NAME)" \
		$(PACKER_CONF_FILE)

publish-ovf-image-req:
	test -n "$(GITHUB_TOKEN)"
	test -n "$(GITHUB_USER)"
	test -n "$(GITHUB_REPOSITORY_NAME)"
	test -n "$(GITHUB_TAG)"

publish-ovf-image: publish-ovf-image-req dist-ovf-image
	ovf_dir="$(RELEASE_PACKAGE_NAME)-ovf-vmdk"
	ovf_artifact=${ovf_dir}.tgz
	tar zcf ${ovf_artifact} ${ovf_dir}/
	gh_release_cmd=linux-amd64-github-release
	gh_release_cmd_bz2=${gh_release_cmd}.bz2
	gh_release_version=v0.9.0
	wget -O ${gh_release_cmd_bz2} \
			https://github.com/github-release/github-release/releases/download/${gh_release_version}/${gh_release_cmd_bz2}
	bunzip2 ${gh_release_cmd_bz2}
	chmod +x ./${gh_release_cmd}
	./${gh_release_cmd} upload \
		--user $GITHUB_USER \
		--repo "${GITHUB_REPOSITORY_NAME}" \
		--tag ${GITHUB_TAG} \
		--name "${ovf_dir}" \
		--file ./${ovf_artifact}		

dist-cloud-image: dist-cloud-image-aws dist-cloud-image-gcp dist-cloud-image-azure