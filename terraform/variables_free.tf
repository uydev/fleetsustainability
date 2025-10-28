variable "mongo_uri" {
  type        = string
  description = "MongoDB Atlas connection string"
  sensitive   = true
}

variable "jwt_secret" {
  type        = string
  description = "JWT secret for authentication"
  sensitive   = true
}

