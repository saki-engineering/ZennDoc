resource "aws_ecs_service" "myecs" {
  name    = join("-", [var.base_name, "service"])
  cluster = aws_ecs_cluster.myecs.id

  task_definition = aws_ecs_task_definition.myecs.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  depends_on = [aws_lb_listener.myecs]

  load_balancer {
    target_group_arn = aws_lb_target_group.myecs.arn
    container_name   = "gRPC-server"
    container_port   = 8080
  }

  network_configuration {
    subnets          = data.aws_subnets.myecs_private.ids
    security_groups  = [aws_security_group.myecs_service.id]
    assign_public_ip = false
  }
}

resource "aws_security_group" "myecs_service" {
  name   = join("-", [var.base_name, "service", "sg"])
  vpc_id = data.aws_vpc.myecs.id

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port       = 8080
    to_port         = 8080
    protocol        = "tcp"
    security_groups = [aws_security_group.myecs_alb.id]
  }
}