output "instance_public_ip" {
  value       = aws_instance.fleet_backend.public_ip
  description = "Public IP address of the EC2 instance"
}

output "instance_public_dns" {
  value       = aws_instance.fleet_backend.public_dns
  description = "Public DNS name of the EC2 instance"
}

output "backend_url" {
  value       = "http://${aws_instance.fleet_backend.public_ip}:8080"
  description = "Backend API URL (use this for frontend Netlify env vars)"
}

output "ssh_command" {
  value       = "ssh -i your-key.pem ubuntu@${aws_instance.fleet_backend.public_ip}"
  description = "SSH command to access the instance (use your private key)"
}

output "current_account_id" {
  value       = data.aws_caller_identity.current.account_id
  description = "AWS Account ID that Terraform deployed to"
}

