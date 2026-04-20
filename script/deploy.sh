#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TERRAFORM_DIR="$PROJECT_ROOT/terraform"
FRONTEND_DIR="$PROJECT_ROOT/src/frontend"

echo "=== Deploying MissionProgress Service ==="

# Step 1: Build Lambda functions + Frontend
echo ""
echo "--- Step 1: Building Lambda functions + Frontend ---"
bash "$SCRIPT_DIR/build.sh"

# Step 2: Terraform init & apply
echo ""
echo "--- Step 2: Terraform init ---"
cd "$TERRAFORM_DIR"
terraform init

echo ""
echo "--- Step 3: Terraform apply ---"
terraform apply -auto-approve

# Step 4: Upload frontend to S3
echo ""
echo "--- Step 4: Uploading frontend to S3 ---"
FRONTEND_BUCKET=$(terraform output -raw frontend_bucket)
aws s3 sync "$FRONTEND_DIR/out/" "s3://$FRONTEND_BUCKET/" --delete \
  --cache-control "public, max-age=3600"

echo ""
echo "=== Deployment complete ==="
echo ""
echo "--- Outputs ---"
terraform output
echo ""
echo "--- Frontend URL ---"
echo "http://$(terraform output -raw frontend_url)"
echo ""
echo "--- API Key ---"
terraform output -raw api_key_value