## Lambda function for monitoring AWS service usage with Anodot

## Supported services:
- EC2
- EBS
- S3
- ELB 

## Installation and package build
---
Build and installation are performed with make tool via Docker (Docker Engine must be avilable).

All neccessary tasks are described in Makefile. 

Run make help to see all available tasks
```bash
make help
 Available tasks:
	 build-image     -- build image usage_lambda:1.0 with all necessary dependencies for lambda function build and lamdba function creation
	 build           -- will build source code. Lambda function binary name usage_lambda
	 create-archive  -- will create archive with binary ready to upload on S3
	 clean           -- will delete archive and binary
	 make copy_to_s3 LAMBDA_S3=your-bucket-name           -- copy lambda archive to s3
	 make copy_config_s3 LAMBDA_S3=your-bucket-name       -- copy config file to s3
	 clean-image     -- will delete usage_lambda image
	 deploy          -- will run build-image, build, build-image, copy_to_s3

 Terraform related tasks:
	 terraform-init     -- will initialize terraform providers and modules
	 terraform-plan     -- will create an execution plan. Shows what will done. What services will be created
	 terraform-apply    -- will apply an execution plan.
	 terraform-plan-destroy    -- will create plan of destroying lambda function.
	 terraform-apply-destroy   -- will destroy lambda functions.
	 create-function           -- will run  terraform-init, terraform-plan, terraform-apply .
```

To upload function to aws need to create zip arhive with binary file. 

For creation neccessary infratructure used terraform (https://www.terraform.io/docs/index.html)

### Installation steps
---
For installation you should have make tool installed on your PC and set AWS_DEFAULT_REGION, AWS_SECRET_ACCESS_KEY, AWS_ACCESS_KEY_ID env vars.

Steps to create and deploy lambda functions:

1. Build and upload lambda binary:

``` bash
make deploy LAMBDA_S3=your-bucket-name
```

2.  Fill terraform/input.tfvars with your data. This is file is needed by terraform and store terraform vars
``` bash 
cat input.tfvars
# Token of anodot customer
token     =
# Url to anodot
anodotUrl =
# s3 bucket where lambda function stored
s3_bucket =

# Regions where metrics will be fetched:
regions = ["region1", "region2"]
```
Please notice that for each region will be created separate function (it will be fetching metric for this region) but it will be deployed into AWS_DEFAULT_REGION. 


3. Deploy lambda function into AWS

```bash
make create-function
```

Please be aware that terraform will create a state file in terraform/ directory. State is hihgly important for future updates and destroy infrastructure.

### How to destroy lambda functions ?
---
``` bash
make terraform-plan-destroy -- to create plan 

make terraform-apply-destroy -- to apply destroy
```
### If we want to deploy function in multiple accounts, how we can distionguish metrics ?

Add variable accountId into input.tfvars file and to your metrics will be added property account_id.

