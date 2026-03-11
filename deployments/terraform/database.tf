# RDS PostgreSQL
resource "aws_db_subnet_group" "main" {
  name       = "auth-db-${var.environment}"
  subnet_ids = aws_subnet.private[*].id

  tags = {
    Name = "auth-db-subnet-${var.environment}"
  }
}

resource "aws_db_instance" "main" {
  identifier     = "auth-db-${var.environment}"
  engine         = "postgres"
  engine_version = "16.3"
  instance_class = var.db_instance_class

  db_name  = var.db_name
  username = var.db_username
  password = random_password.db_password.result

  allocated_storage     = 20
  max_allocated_storage = 100
  storage_encrypted     = true

  db_subnet_group_name   = aws_db_subnet_group.main.name
  vpc_security_group_ids = [aws_security_group.db.id]

  multi_az            = var.environment == "production"
  skip_final_snapshot = var.environment != "production"

  backup_retention_period = var.environment == "production" ? 7 : 1

  tags = {
    Environment = var.environment
  }
}

resource "random_password" "db_password" {
  length  = 32
  special = false
}

# ElastiCache Redis
resource "aws_elasticache_subnet_group" "main" {
  name       = "auth-redis-${var.environment}"
  subnet_ids = aws_subnet.private[*].id
}

resource "aws_elasticache_cluster" "main" {
  cluster_id           = "auth-redis-${var.environment}"
  engine               = "redis"
  engine_version       = "7.1"
  node_type            = var.redis_node_type
  num_cache_nodes      = 1
  parameter_group_name = "default.redis7"

  subnet_group_name  = aws_elasticache_subnet_group.main.name
  security_group_ids = [aws_security_group.redis.id]

  tags = {
    Environment = var.environment
  }
}

# Store secrets in SSM Parameter Store
resource "random_password" "jwt_secret" {
  length  = 64
  special = false
}

resource "aws_ssm_parameter" "jwt_secret" {
  name  = "/auth-service/${var.environment}/jwt-secret"
  type  = "SecureString"
  value = random_password.jwt_secret.result

  tags = {
    Environment = var.environment
  }
}

resource "aws_ssm_parameter" "db_url" {
  name  = "/auth-service/${var.environment}/database-url"
  type  = "SecureString"
  value = "postgres://${var.db_username}:${random_password.db_password.result}@${aws_db_instance.main.endpoint}/${var.db_name}?sslmode=require"

  tags = {
    Environment = var.environment
  }
}
