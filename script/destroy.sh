#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TERRAFORM_DIR="$PROJECT_ROOT/terraform"

echo "=== Destroying MissionProgress Service ==="

cd "$TERRAFORM_DIR"
terraform destroy -auto-approve

echo ""
echo "=== Destroy complete ==="
