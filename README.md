## Lamda function for monitoring AWS service usage with Anodot

## Supported services:
- EC2
- EBS
- S3
- ELB 

## Installation and package build

Build and installation are performed with make tool.

All neccessary tasks are described in Makefile. 

Run make help to see all available tasks
```bash
make help
 Available tasks:
	 build-image     -- build image usage_lambda:1.0 with all necessary dependencies for lambda function build and lamdba function creation
	 build           -- will build source code. Lambda function binary name usage_lambda
	 create-archive  -- will create archive with binary ready to upload on S3
	 clean           -- will delete archive and binary
	 clean-image     -- will delete usage_lambda image

 Terraform related tasks:
	 terraform-init     -- will initialize terraform providers and modules
	 terraform-plan     -- will create an execution plan. Shows what will done. What services will be created
	 terraform-apply     -- will apply an execution plan.
```

To upload function to aws need to create zip arhive with binary file. 

For creation neccessary infratructure used terraform (https://www.terraform.io/docs/index.html)

### Installation steps
For installation you should have make tool installed on your PC and set AWS_DEFAULT_REGION, AWS_SECRET_ACCESS_KEY ,AWS_ACCESS_KEY_ID env vars.

1. Run **make build-image** to build image with all dependencies for golang and terraform binaries

2. Run **make build** to build lambda binary

3. Run **make create-archive** to create archive with bynaries 

4. Run **make copy_to_s3 LAMBDA_S3=your-bucket-name** to upload arhive to s3 where lambda will be stored

5. Fill terraform/input.tfvars with your data 

6. Run **make terraform-init**

7. Run **make terraform-plan**

8. Run **make terraform-apply**

Please be aware that terraform will create a state file in terraform/ directory.
