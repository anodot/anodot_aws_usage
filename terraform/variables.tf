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

variable "env_name" {
    type = string
    description = "Name of current env. Will be used as part of resource name."
}
