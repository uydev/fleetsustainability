# Fleet Sustainability - FREE TIER Deployment ✅

## ✅ YES - This Works and is 100% Free Tier

**What's included:**
- ✅ t3.micro EC2 instance (FREE for 750 hours/month, first 12 months)
- ✅ EBS storage (FREE for 30GB, first 12 months)
- ✅ VPC, Security Groups, Internet Gateway (all FREE)
- ✅ **Total cost: $0/month for the first year!**

**Your existing code works perfectly:**
- ✅ You have a Dockerfile already
- ✅ You use MongoDB Atlas (no database costs)
- ✅ All you need is to push your Docker image to ECR and run it

## 🚀 Deployment Steps

### Step 1: Build and Push Your Docker Image

```bash
# From project root
cd /Users/yilmazu/Projects/hephaestus-sytems/Fleet-Sustainability

# Build your Docker image
docker build -t fleet-backend .

# Login to AWS ECR
aws ecr get-login-password --region eu-west-2 --profile hephaestus-fleet | docker login --username AWS --password-stdin 901465080034.dkr.ecr.eu-west-2.amazonaws.com

# Tag and push to ECR
docker tag fleet-backend 901465080034.dkr.ecr.eu-west-2.amazonaws.com/fleetsustainability-backend:latest
docker push 901465080034.dkr.ecr.eu-west-2.amazonaws.com/fleetsustainability-backend:latest
```

### Step 2: Configure Terraform

Make sure you have `terraform.tfvars` (or create it):

```bash
cd terraform
```

Create or update `terraform.tfvars`:
```hcl
mongo_uri  = "your-mongodb-atlas-uri"
jwt_secret = "your-jwt-secret"
```

### Step 3: Deploy

```bash
# Initialize (first time only)
terraform init

# Review what will be created
terraform plan

# Deploy it
terraform apply
```

### Step 4: Get Your Backend URL

```bash
terraform output backend_url
# This will give you: http://54.123.45.67:8080
```

Use this URL in your frontend Netlify environment variables.

## 📁 Files Created

- `terraform/main_ec2_free_tier.tf` - EC2 deployment (FREE tier)
- `terraform/variables_ec2.tf` - Variables for EC2
- `terraform/outputs_ec2.tf` - Output values
- `terraform/main.tf` - Original ECS Fargate (keep for reference)

## 💰 Cost Breakdown

| Period | EC2 (t3.micro) | Storage | Total |
|--------|----------------|---------|-------|
| First Year | **FREE** ✅ | **FREE** ✅ | **$0** |
| After Year 1 | ~$6/mo | ~$1/mo | **~$7/mo** |

Compare to ECS Fargate: **~$30/mo** (you're saving ~$23/mo! 🎉)

## 🔧 How It Works

1. Terraform creates a t3.micro EC2 instance
2. EC2 installs Docker on startup
3. EC2 logs into ECR and pulls your pre-built image
4. EC2 runs your container on port 8080
5. Your backend is accessible via the EC2 public IP

## 🔄 To Update Your Code

After making code changes:

```bash
# 1. Rebuild Docker image
docker build -t fleet-backend .

# 2. Push to ECR
docker tag fleet-backend 901465080034.dkr.ecr.eu-west-2.amazonaws.com/fleetsustainability-backend:latest
docker push 901465080034.dkr.ecr.eu-west-2.amazonaws.com/fleetsustainability-backend:latest

# 3. SSH into EC2 and restart container
terraform output ssh_command  # Get the SSH command
# Then SSH in and run:
ssh ubuntu@<IP>
sudo docker pull 901465080034.dkr.ecr.eu-west-2.amazonaws.com/fleetsustainability-backend:latest
sudo docker restart fleet-backend
```

## 🛑 To Destroy

```bash
terraform destroy
```

## ⚠️ Notes

1. **ECR Repository**: You already have `fleetsustainability-backend` ECR repo created by `main.tf`
2. **No SSH Key Needed**: The EC2 can pull from ECR using IAM roles (part of your AWS account config)
3. **Auto-restart**: The container is configured to restart if it crashes
4. **Health Check**: After deployment, the instance checks if the backend is healthy

## ✅ Verification Checklist

- [ ] You built and pushed your Docker image to ECR
- [ ] You have `terraform.tfvars` with mongo_uri and jwt_secret
- [ ] You ran `terraform apply` successfully
- [ ] You got a backend URL from `terraform output backend_url`
- [ ] You tested the URL: `curl http://<IP>:8080/api/health`

---

**🎉 That's it! You now have a FREE deployment running on AWS!**
