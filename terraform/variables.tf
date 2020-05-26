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

variable "accountId" {
    type = string
    description = "Custom Id in order to distinguish different AWS accounts"
    default = ""
}

variable "function_id" {
    type = string
    description = "Custom string which will be used like a prefix of the lambda function name"
}