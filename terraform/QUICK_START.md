# Quick Start - Free Tier Deployment

## ✅ YES - This is ready to use and 100% FREE!

### What You Get:
- **EC2 t3.micro**: FREE for 750 hours/month (first year) = **$0/month**
- **EBS Storage**: FREE for 30GB (first year) = **$0/month**
- **Total first year**: **$0/month** ✅
- **After year 1**: ~$7/month (vs $30/month for ECS Fargate)

### 3 Simple Steps:

#### 1️⃣ Build and Push Your Docker Image

```bash
# Go to your project
cd /Users/yilmazu/Projects/hephaestus-sytems/Fleet-Sustainability

# Build the image
docker build -t fleet-backend .

# Login to AWS ECR
aws ecr get-login-password --region eu-west-2 --profile hephaestus-fleet | docker login --username AWS --password-stdin 901465080034.dkr.ecr.eu-west-2.amazonaws.com

# Push to ECR
docker tag fleet-backend 901465080034.dkr.ecr.eu-west-2.amazonaws.com/fleetsustainability-backend:latest
docker push 901465080034.dkr.ecr.eu-west-2.amazonaws.com/fleetsustainability-backend:latest
```

#### 2️⃣ Deploy with Terraform

```bash
cd terraform

# Make sure you have terraform.tfvars with your credentials
cat > terraform.tfvars <<EOF
mongo_uri  = "your-mongodb-atlas-uri"
jwt_secret = "your-jwt-secret"
EOF

# Deploy
terraform init
terraform plan
terraform apply
```

#### 3️⃣ Get Your Backend URL

```bash
terraform output backend_url
# Example output: http://54.123.45.67:8080
```

Use this URL in your frontend Netlify environment variables!

---

## 📁 Files to Use

✅ **Use this**: `terraform/main_ec2_free_tier.tf`  
✅ **Variables**: `terraform/variables_ec2.tf`  
✅ **Outputs**: `terraform/outputs_ec2.tf`

⚠️ **Don't use**: `terraform/main.tf` (this is for ECS Fargate and costs money)

---

## 🎯 What Happens

1. Terraform creates a **t3.micro EC2 instance** (FREE tier)
2. The instance installs Docker
3. The instance pulls your Docker image from ECR
4. The instance runs your container on port 8080
5. Your backend is live! 🎉

---

## 🔄 Update Your Code Later

```bash
# 1. Rebuild
docker build -t fleet-backend .
docker tag fleet-backend 901465080034.dkr.ecr.eu-west-2.amazonaws.com/fleetsustainability-backend:latest
docker push 901465080034.dkr.ecr.eu-west-2.amazonaws.com/fleetsustainability-backend:latest

# 2. SSH and restart
# Get the IP from terraform output instance_public_ip
ssh ubuntu@<IP>

# On the EC2 instance:
sudo docker pull 901465080034.dkr.ecr.eu-west-2.amazonaws.com/fleetsustainability-backend:latest
sudo docker restart fleet-backend
```

---

## ✨ That's It!

Your Docker container is now running on AWS **completely free** for the first year!

