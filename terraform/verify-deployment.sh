#!/bin/bash
set -e

echo "🔍 Verifying AWS configuration for Fleet Sustainability..."
echo ""

AWS_PROFILE="hephaestus-fleet"
export AWS_PROFILE

echo "Using AWS profile: $AWS_PROFILE"
echo ""

# Verify account
CURRENT_ACCOUNT=$(aws sts get-caller-identity --query Account --output text)
EXPECTED_ACCOUNT="901465080034"

echo "Current Account: $CURRENT_ACCOUNT"
echo "Expected Account: $EXPECTED_ACCOUNT"
echo ""

if [ "$CURRENT_ACCOUNT" != "$EXPECTED_ACCOUNT" ]; then
  echo "❌ ERROR: Wrong AWS account!"
  echo "You're about to deploy to the WRONG account!"
  echo "Current:  $CURRENT_ACCOUNT"
  echo "Expected: $EXPECTED_ACCOUNT"
  exit 1
fi

echo "✓ Account verified: $CURRENT_ACCOUNT"
echo "✓ Region: $(aws configure get region --profile $AWS_PROFILE)"
echo ""

# Get IAM user info
CURRENT_USER=$(aws sts get-caller-identity --query Arn --output text)
echo "Current IAM User/Role: $CURRENT_USER"
echo ""

echo "✅ All checks passed! Safe to proceed with terraform."
echo ""

