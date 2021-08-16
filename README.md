## Lambda function for monitoring AWS service usage with Anodot

## Supported services:
- EC2
- EBS
- S3
- ELB 
- NatGateway
- Efs
- DynamoDB
- Cloudfront

## Installation and package build
---
Some notes before you start: 
- Building and installation of the lambda file is performed with the make tool via Docker. Please make sure you have a Docker Engine avilable.
- For creation of the neccessary infratructure we will use terraform (https://www.terraform.io/docs/index.html)
- All neccessary tasks are described in the Makefile below. 
- To upload the lambda function to AWS you need to create a zip archive with the binary file you've built
- You should have access to AWS with minimum following permission:
``` json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "VisualEditor0",
            "Effect": "Allow",
            "Action": [
                "events:DeleteRule",
                "events:PutTargets",
                "events:DescribeRule",
                "events:PutRule",
                "events:ListTagsForResource",
                "events:RemoveTargets",
                "events:ListTargetsByRule",
                "ec2:Describe*",
                "iam:GetRole",
                "iam:GetPolicyVersion",
                "iam:GetPolicy",
                "iam:DeletePolicy",
                "iam:CreateRole",
                "iam:DeleteRole",
                "iam:AttachRolePolicy",
                "iam:CreatePolicy",
                "iam:ListInstanceProfilesForRole",
                "iam:PassRole",
                "iam:DetachRolePolicy",
                "iam:ListPolicyVersions",
                "iam:ListAttachedRolePolicies"
                "lambda:*",
                "logs:*",
                "s3:PutObject",
                "s3:GetObject",
                "s3:DeleteObject"
            ],
            "Resource": "*"
        }
    ]
}
``` 
Run "make help" to see all available tasks:

```bash
make help
 Available tasks:
	 make build-image     -- build image usage_lambda:1.0 with all necessary dependencies for lambda function build and lamdba function creation
	 make build-code      -- will build source code. Lambda function binary name usage_lambda
	 make build           -- will run clean build-image and build-code
	 make create-archive  -- will create archive with binary ready to upload on S3
	 make clean           -- will delete archive and binary
	 make copy_to_s3 LAMBDA_S3=your-bucket-name           -- copy lambda archive to s3
	 make copy_config_s3 LAMBDA_S3=your-bucket-name       -- copy config file to s3
	 make clean-image     -- will delete usage_lambda image
	 make deploy LAMBDA_S3=your-bucket-name          -- will run build-image, build, build-image, copy_to_s3

	 make create-config          -- will run command line menu to help build a new config file

 Terraform related tasks:
	 make terraform-init     -- will initialize terraform providers and modules
	 make terraform-plan     -- will create an execution plan. Shows what will done. What services will be created
	 make terraform-apply    -- will apply an execution plan.
	 make terraform-plan-destroy    -- will create plan of destroying lambda function.
	 make terraform-apply-destroy   -- will destroy lambda functions.
	 make create-function           -- will run  terraform-init, terraform-plan, terraform-apply .
``` 

## Installation steps
---
For the installation you should have the make tool installed on your machine and set the following environment vars:

``` 
AWS_DEFAULT_REGION = <Your Region>
AWS_SECRET_ACCESS_KEY = <Your Secret AWS Access Key>
AWS_ACCESS_KEY_ID = <Your AWS Access Key ID>
``` 

Below are the steps required to create and deploy the lambda function:

1. Update **cloudwatch_metrics.yaml** with regions and metrics you need to push. 

Or just run 
```
make create-config
```
This command will show you command line menu for generation default config.

Please notice that metrics for Cloudfront available only at us-east-1, because it is global service. You can use Cloudfront metrics from another region, but you will need manually update config (more details at FAQ section).

2. Build and upload the lambda binary:

``` bash
make deploy LAMBDA_S3=your-bucket-name
```

3.  Fill **terraform/input.tfvars** with your data. This is file is needed by terraform to store terraform vars


``` bash 
# s3 bucket where lambda function stored
s3_bucket = ""

# Custom string for identification of lambda installation. Will be sent to Anodot as account_id dimesnion.
function_id = "fr-testing"

# Regions where metrics will be fetched:
regions = ["eu-central-1"]

anodot_access_key =  
anodot_data_token= 
```
For obtaining your **anodot_access_key** and **anodot_data_token** - Please refer to this [page](https://support.anodot.com/hc/en-us/articles/360002631114-Token-Management-) 

Please notice that for each region a separate function will be created (it will be fetching metrics for this region) but it will be deployed into AWS_DEFAULT_REGION. 



4. Create the lambda function

For running the next command, you will need your the URL through which you access Anodot.


```bash
make create-function
```

Please be aware that terraform will create a state file in the ```terraform/``` directory. The State is highly important for future updates and destroy infrastructure.

## FAQ 
---

### How do I specify the different regions from which you get Cloudfront metrics?
``` yaml
ap-south-1:
  Cloudfront:
    Region: us-east-1 # Add this options in you need metrics from different region
    CloudWatchMetrics:
    - Name: BytesDownloaded
      Id: test1
      Namespace: AWS/CloudFront
      Period: 3600
      Unit: None
      Stat:  Average
```
In the example above cloudfront metrics will be fetched and pushed for us-east-1 and EBS and EC2 for ap-south-1

### How do I destroy lambda functions ?
---
``` bash
make terraform-plan-destroy -- to create plan 

make terraform-apply-destroy -- to apply destroy
```
### If I want to deploy a function in multiple accounts, how can I distionguish between the metrics ?

Add variable accountId into **input.tfvars** file and to your metrics will be added property account_id.

### List of custom metrics:
A custom metric is a metric calculated directly by the lambda function (not fetched from CLoudwatch)

EBS has custom metric: Size

EC2 has: CoreCount and VCpuCount - cores count with hyperthreading 

Efs has: 
- Size_All - The latest known metered size (in bytes) of data stored in the file system.
- Size_Infrequent - The latest known metered size (in bytes) of data stored in the Infrequent Access storage class.
- Size_Standard - The latest known metered size (in bytes) of data stored in the Standard storage class

### How do I configure which metrics are pushed per region ?
Each region should have a separate section in cloudwatch_metrics.yaml file with list of metrics to be fetched: 
```yaml
us-east-1: # Region where lambda supposed to fetch metrics
  Cloudfront:
    CloudWatchMetrics:
    - Name: BytesDownloaded
      Id: test1
      Namespace: AWS/CloudFront
      Period: 3600
      Unit: None
      Stat:  Average
ap-south-1:
  EBS:
    CustomMetrics:
      - Size
  EC2:
    CustomMetrics:
      - CoreCount
```

