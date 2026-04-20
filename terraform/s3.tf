# ---------------------------------------------------
# S3 Bucket: Evidence Storage
# ---------------------------------------------------
resource "aws_s3_bucket" "evidence" {
  bucket = "${var.project_name}-evidence-${data.aws_caller_identity.current.account_id}"

  tags = {
    Project = var.project_name
  }
}

# ---------------------------------------------------
# CORS Configuration (presigned URL upload)
# ---------------------------------------------------
resource "aws_s3_bucket_cors_configuration" "evidence_cors" {
  bucket = aws_s3_bucket.evidence.id

  cors_rule {
    allowed_headers = ["*"]
    allowed_methods = ["PUT"]
    allowed_origins = ["*"]
    max_age_seconds = 3600
  }
}

# ---------------------------------------------------
# Block Public Access
# ---------------------------------------------------
resource "aws_s3_bucket_public_access_block" "evidence_public_access" {
  bucket = aws_s3_bucket.evidence.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}
