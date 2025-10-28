# Fleet Sustainability - Terraform Deployment

Deploy the Fleet Sustainability backend to AWS ECS Fargate.

## Quick Start

1. **Verify deployment safety:**
   ```bash
   chmod +x verify-deployment.sh
   ./verify-deployment.sh
   ```

2. **Initialize Terraform:**
   ```bash
   terraform init
   ```

3. **Plan deployment:**
   ```bash
   terraform plan
   ```

4. **Apply (if plan looks good):**
   ```bash
   terraform apply
   ```

5. **Get backend URL:**
   ```bash
   terraform output load_balancer_dns
   ```

## Destroy Resources

```bash
terraform destroy
```

## Outputs

- `load_balancer_dns` - Backend API URL (use this for frontend Netlify env vars)
- `ecr_repository_url` - ECR repository URL
- `current_account_id` - AWS Account ID (should be 901465080034)
- `ecs_cluster_name` - ECS Cluster name
- `ecs_service_name` - ECS Service name

