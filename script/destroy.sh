#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TERRAFORM_DIR="$PROJECT_ROOT/terraform"

echo "=== Destroying MissionProgress Service ==="

cd "$TERRAFORM_DIR"

# Empty S3 buckets before destroy (required for deletion)
echo "--- Emptying S3 buckets ---"
FRONTEND_BUCKET=$(terraform output -raw frontend_bucket 2>/dev/null || true)
if [ -n "$FRONTEND_BUCKET" ]; then
  aws s3 rm "s3://$FRONTEND_BUCKET" --recursive || true
fi

EVIDENCE_BUCKET=$(terraform output -raw evidence_bucket 2>/dev/null || true)
if [ -n "$EVIDENCE_BUCKET" ]; then
  aws s3 rm "s3://$EVIDENCE_BUCKET" --recursive || true
fi

echo "--- Terraform destroy ---"
terraform destroy -auto-approve

echo ""
echo "=== Destroy complete ==="
