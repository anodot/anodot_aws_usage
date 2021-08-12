variable "s3_bucket" {
    type = string
    description = "S3 bucket where lambda function will be stored"
}

variable "regions" {
    type  = list(string)
    description = "List of regions where lambda will fetch data. Will be created lambda per region"
}

variable "function_id" {
    type = string
    description = "Custom string for distinguishing different lambda installation"
}

variable "anodot_access_key" {
    type = string
    description = "Anodot API access key. Terraform needs this to create secret in AWS Secrets Manager"
}

variable "anodot_data_token" {
    type = string
    description = "Anodot data collection token. Terraform needs this to create secret in AWS Secrets Manager"
}