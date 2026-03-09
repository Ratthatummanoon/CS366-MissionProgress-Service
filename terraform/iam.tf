data "aws_iam_role" "lab_role" {
  name = "LabRole"
}

locals {
  lab_role_arn = var.lab_role_arn != "" ? var.lab_role_arn : data.aws_iam_role.lab_role.arn
}
