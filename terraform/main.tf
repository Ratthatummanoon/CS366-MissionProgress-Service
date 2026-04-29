terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
  required_version = ">= 1.0"
}

locals {
  api_key_value = var.api_key_value
}

provider "aws" {
  region = var.aws_region
}

data "aws_caller_identity" "current" {}
