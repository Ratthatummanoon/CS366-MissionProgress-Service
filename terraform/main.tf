terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
    }
  }
  required_version = ">= 1.0"
}

resource "random_password" "api_key" {
  length  = 32
  special = false
}

locals {
  api_key_value = var.api_key_value != "" ? var.api_key_value : random_password.api_key.result
}

provider "aws" {
  region = var.aws_region
}

data "aws_caller_identity" "current" {}
