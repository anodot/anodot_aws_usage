variable "anodotUrl" {
    type = string
    description = "Anodot Url" 
}

variable "token" {
    type = string
    description = "Anodot token"
}

variable "s3_bucket" {
    type = string
    description = "S3 bucket where lambda function will be stored"
}

variable "regions" {
    type  = list(string)
    description = "List of regions where lambda will fetch data. Will be created lambda per region"
}

variable "env_name" {
    type = string
    description = "Env name will be used as prefix name of resources"
}