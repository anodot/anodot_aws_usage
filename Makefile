BUILD_IMAGE := usage_lambda
BUILD_IMAGE_VERSION := 1.0

CONTAINER_BASH := docker run --workdir /output -v "$(PWD)":/output "$(BUILD_IMAGE)":$(BUILD_IMAGE_VERSION)
GO :=  $(CONTAINER_BASH) go

TERRAFORM_CMD := docker run -e AWS_DEFAULT_REGION  -e AWS_SECRET_ACCESS_KEY -e AWS_ACCESS_KEY_ID --workdir /output/terraform/ -v "$(PWD)":/output "$(BUILD_IMAGE)":$(BUILD_IMAGE_VERSION) terraform 
GOFLAGS=-mod=vendor

AWSCLI := docker run -e AWS_DEFAULT_REGION  -e AWS_SECRET_ACCESS_KEY -e AWS_ACCESS_KEY_ID --workdir /output -v "$(PWD)":/output "$(BUILD_IMAGE)":$(BUILD_IMAGE_VERSION) aws 
GOARCH := amd64
GOOS := linux

GOLINT_VERSION:=1.23.1

BUILD_FLAGS = GO111MODULE=on CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) GOFLAGS=$(GOFLAGS)
APPLICATION_NAME := usage_lambda
LAMBDA_ARCHIVE := usage_lambda.zip

GREEN := \033[0;32m
NC := \033[0m
CYAN := \033[0;36m

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

terraform-apply:
	$(TERRAFORM_CMD) apply "start"

copy_to_s3:
	$(AWSCLI) s3 cp $(LAMBDA_ARCHIVE) s3://$(LAMBDA_S3) 

help:
	@echo "$(CYAN) Available tasks: $(NC)"
	@echo "	$(GREEN) build-image $(NC)    -- build image $(BUILD_IMAGE):$(BUILD_IMAGE_VERSION) with all necessary dependencies for lambda function build and lamdba function creation"
	@echo "	$(GREEN) build $(NC)          -- will build source code. Lambda function binary name $(APPLICATION_NAME)"
	@echo "	$(GREEN) create-archive $(NC) -- will create archive with binary ready to upload on S3"
	@echo "	$(GREEN) clean $(NC)          -- will delete archive and binary"
	@echo "	$(GREEN) make copy_to_s3 LAMBDA_S3=your-bucket-name $(NC)          -- copy lambda archive to s3"
	@echo "	$(GREEN) clean-image $(NC)    -- will delete $(BUILD_IMAGE) image \n"

	@echo "$(CYAN) Terraform related tasks: $(NC) "
	@echo "	$(GREEN) terraform-init $(NC)    -- will initialize terraform providers and modules "
	@echo "	$(GREEN) terraform-plan $(NC)    -- will create an execution plan. Shows what will done. What services will be created"
	@echo "	$(GREEN) terraform-apply $(NC)   -- will apply an execution plan."