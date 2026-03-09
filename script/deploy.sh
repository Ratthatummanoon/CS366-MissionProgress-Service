#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TERRAFORM_DIR="$PROJECT_ROOT/terraform"

echo "=== Deploying MissionProgress Service ==="

# Step 1: Build Lambda functions
echo ""
echo "--- Step 1: Building Lambda functions ---"
bash "$SCRIPT_DIR/build.sh"

# Step 2: Terraform init & apply
echo ""
echo "--- Step 2: Terraform init ---"
cd "$TERRAFORM_DIR"
terraform init

echo ""
echo "--- Step 3: Terraform apply ---"
terraform apply -auto-approve

echo ""
echo "=== Deployment complete ==="
echo ""
echo "--- Outputs ---"
terraform output
echo "see api_key_value for the API key to use with Postman or curl"
cd "$TERRAFORM_DIR"
terraform output -raw api_key_value