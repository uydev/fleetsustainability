terraform {
  required_version = ">= 1.0"
  
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    null = {
      source  = "hashicorp/null"
      version = "~> 3.0"
    }
  }
}

provider "aws" {
  profile = "hephaestus-fleet"
  region  = "eu-west-2"
  
  default_tags {
    tags = {
      Project     = "Fleet-Sustainability"
      ManagedBy   = "Terraform"
      Environment = "Production"
      Owner       = "Hephaestus-Systems"
    }
  }
}

# Safety check - verify we're using the right account
data "aws_caller_identity" "current" {}

# Verify account before creating resources
resource "null_resource" "verify_account" {
  provisioner "local-exec" {
    command = <<-EOT
      EXPECTED_ACCOUNT="901465080034"
      ACTUAL_ACCOUNT="${data.aws_caller_identity.current.account_id}"
      if [ "$EXPECTED_ACCOUNT" != "$ACTUAL_ACCOUNT" ]; then
        echo "ERROR: Wrong AWS account! Expected $EXPECTED_ACCOUNT but got $ACTUAL_ACCOUNT"
        exit 1
      fi
      echo "✓ Deploying to correct account: $ACTUAL_ACCOUNT"
    EOT
  }
  
  triggers = {
    always_run = timestamp()
  }
}

# VPC for EC2
resource "aws_vpc" "fleet_vpc" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true
  
  tags = {
    Name = "fleet-sustainability-vpc"
  }
  
  depends_on = [null_resource.verify_account]
}

# Internet Gateway
resource "aws_internet_gateway" "fleet_igw" {
  vpc_id = aws_vpc.fleet_vpc.id
  
  tags = {
    Name = "fleet-sustainability-igw"
  }
}

# Subnet
resource "aws_subnet" "fleet_public" {
  vpc_id                  = aws_vpc.fleet_vpc.id
  cidr_block              = "10.0.1.0/24"
  availability_zone       = data.aws_availability_zones.available.names[0]
  map_public_ip_on_launch = true
  
  tags = {
    Name = "fleet-sustainability-public"
  }
}

data "aws_availability_zones" "available" {
  state = "available"
}

# Route Table
resource "aws_route_table" "fleet_rt" {
  vpc_id = aws_vpc.fleet_vpc.id
  
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.fleet_igw.id
  }
  
  tags = {
    Name = "fleet-sustainability-rt"
  }
}

resource "aws_route_table_association" "fleet_rta" {
  subnet_id      = aws_subnet.fleet_public.id
  route_table_id = aws_route_table.fleet_rt.id
}

# Security Group for EC2
resource "aws_security_group" "fleet_sg" {
  name        = "fleet-sustainability-ec2-sg"
  description = "Security group for Fleet Sustainability Backend on EC2"
  vpc_id      = aws_vpc.fleet_vpc.id
  
  ingress {
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "Backend API"
  }
  
  # SSH access (optional - remove if not needed)
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "SSH"
  }
  
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
    description = "Allow all outbound"
  }
  
  tags = {
    Name = "fleet-sustainability-ec2-sg"
  }
}

# EC2 Instance (t3.micro - Free Tier eligible)
resource "aws_instance" "fleet_backend" {
  ami           = data.aws_ami.ubuntu.id
  instance_type = "t3.micro"
  
  subnet_id              = aws_subnet.fleet_public.id
  vpc_security_group_ids = [aws_security_group.fleet_sg.id]
  iam_instance_profile   = aws_iam_instance_profile.ec2_profile.name
  
  # Enable automatic public IP
  associate_public_ip_address = true
  
  # User data to install Docker and run the container
  user_data = <<-EOF
#!/bin/bash
set -e

echo "=== Installing Docker and AWS CLI ==="
export DEBIAN_FRONTEND=noninteractive
apt-get update -y
apt-get install -y docker.io wget awscli

# Start Docker
systemctl start docker
systemctl enable docker

echo "=== Logging into ECR ==="
# Login to ECR using IAM role
aws ecr get-login-password --region eu-west-2 | docker login --username AWS --password-stdin 901465080034.dkr.ecr.eu-west-2.amazonaws.com

echo "=== Pulling and running container ==="
# Pull and run the pre-built image
docker run -d \
  --name fleet-backend \
  --restart unless-stopped \
  -p 8080:8080 \
  -e MONGO_URI="${var.mongo_uri}" \
  -e MONGO_DB=fleet \
  -e JWT_SECRET="${var.jwt_secret}" \
  -e TELEMETRY_TTL_DAYS=30 \
  -e WEBSOCKETS_ENABLED=1 \
  -e PORT=8080 \
  901465080034.dkr.ecr.eu-west-2.amazonaws.com/fleetsustainability-backend:latest

echo "=== Container started successfully ==="
EOF
  
  tags = {
    Name = "fleet-sustainability-backend"
  }

  lifecycle {
    ignore_changes = [user_data]
  }
}

# Get latest Ubuntu 22.04 LTS AMI
data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"] # Canonical

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

# CloudWatch Log Group
resource "aws_cloudwatch_log_group" "fleet_logs" {
  name              = "/ec2/fleet-sustainability"
  retention_in_days = 7
}

# IAM role for EC2 to access ECR
resource "aws_iam_role" "ec2_role" {
  name = "fleet-ec2-role"
  
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "ec2.amazonaws.com"
      }
    }]
  })
  
  # Disable tags to avoid needing iam:TagRole permission
  tags = {}
}

resource "aws_iam_role_policy_attachment" "ec2_ecr_access" {
  role       = aws_iam_role.ec2_role.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
}

resource "aws_iam_instance_profile" "ec2_profile" {
  name = "fleet-ec2-profile"
  role = aws_iam_role.ec2_role.name
  
  # Disable tags to avoid needing iam:TagInstanceProfile permission
  tags = {}
}

