variable "aws_region" {
  description = "AWS region for all resources"
  type        = string
  default     = "eu-central-1"
}

variable "environment" {
  description = "Environment name (dev, staging, production)"
  type        = string
  default     = "dev"
}

variable "db_instance_class" {
  description = "RDS instance class"
  type        = string
  default     = "db.t3.micro"
}

variable "db_name" {
  description = "Database name"
  type        = string
  default     = "auth"
}

variable "db_username" {
  description = "Database master username"
  type        = string
  default     = "auth"
  sensitive   = true
}

variable "redis_node_type" {
  description = "ElastiCache node type"
  type        = string
  default     = "cache.t3.micro"
}

variable "app_image" {
  description = "Docker image for the auth service"
  type        = string
}

variable "app_cpu" {
  description = "ECS task CPU units"
  type        = number
  default     = 256
}

variable "app_memory" {
  description = "ECS task memory (MB)"
  type        = number
  default     = 512
}
