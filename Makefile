BUILD_IMAGE := usage_lambda
BUILD_IMAGE_VERSION := 1.0

CONTAINER_BASH := docker run --workdir /output -v "$(PWD)":/output "$(BUILD_IMAGE)":$(BUILD_IMAGE_VERSION)
GO :=  $(CONTAINER_BASH) go

TERRAFORM_CMD := docker run -e AWS_DEFAULT_REGION  -e AWS_SECRET_ACCESS_KEY -e AWS_ACCESS_KEY_ID --workdir /output/terraform/ -v "$(PWD)":/output "$(BUILD_IMAGE)":$(BUILD_IMAGE_VERSION) terraform 
GOFLAGS=-mod=vendor

GOARCH := amd64
GOOS := linux

GOLINT_VERSION:=1.23.1

BUILD_FLAGS = GO111MODULE=on CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) GOFLAGS=$(GOFLAGS)
APPLICATION_NAME := usage_lambda
LAMBDA_ARCHIVE := usage_lambda.zip

clean-image:
	docker rmi -f `docker images $(BUILD_IMAGE):$(BUILD_IMAGE_VERSION) -a -q` || true

clean:
	@rm -rf $(APPLICATION_NAME)
	@rm -rf $(LAMBDA_ARCHIVE)

build-image:
	#docker build  -t $(BUILD_IMAGE):$(BUILD_IMAGE_VERSION) src/
	docker build --no-cache -t $(BUILD_IMAGE):$(BUILD_IMAGE_VERSION) .

build:
	@echo ">> building binaries with version $(VERSION)"
	$(BUILD_FLAGS) $(GO)  build -o $(APPLICATION_NAME) 

create-archive:
	$(CONTAINER_BASH) zip $(LAMBDA_ARCHIVE) $(APPLICATION_NAME)

terraform-state-list:
	$(TERRAFORM_CMD) state list

terraform-init:
	$(TERRAFORM_CMD) init

terraform-plan:
	$(TERRAFORM_CMD) plan -out start -var-file input.tfvars