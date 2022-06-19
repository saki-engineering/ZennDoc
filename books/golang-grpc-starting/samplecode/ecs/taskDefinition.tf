# タスク定義
resource "aws_ecs_task_definition" "myecs" {
  family                   = join("-", [var.base_name, "task", "definition"])
  requires_compatibilities = ["FARGATE"]

  network_mode = "awsvpc"
  cpu          = 256
  memory       = 512

  container_definitions = jsonencode([
    {
      name      = "gRPC-server"
      image     = "${data.aws_ecr_repository.myecs.repository_url}:${var.image_tag}"
      essential = true
      portMappings = [
        {
          containerPort = 8080
          hostPort      = 8080
        }
      ]
      logConfiguration = {
        logDriver = "awsfirelens"
        options = {
          Name              = "cloudwatch"
          region            = var.region
          log_group_name    = join("/", ["ecs", var.base_name])
          log_stream_prefix = "grpc"
        }
      }
      healthCheck = {
        command = ["CMD-SHELL", "/bin/grpc_health_probe -addr=:8080 || exit 1"]
      }
    },
    {
      name      = "log-router"
      image     = "public.ecr.aws/aws-observability/aws-for-fluent-bit:stable"
      essential = true

      firelensConfiguration = {
        type = "fluentbit"
        options = {
          enable-ecs-log-metadata = "true"
          config-file-type        = "file"
          config-file-value       = "/fluent-bit/configs/parse-json.conf"
        }
      }
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-region        = var.region
          awslogs-group         = join("/", ["ecs", var.base_name])
          awslogs-stream-prefix = "logger"
        }
      }
    }
  ])

  execution_role_arn = aws_iam_role.myecs_task_execution_role.arn
  task_role_arn      = aws_iam_role.myecs_task_role.arn
}

# execution role
resource "aws_iam_role" "myecs_task_execution_role" {
  name               = join("-", [var.base_name, "execution-role"])
  assume_role_policy = data.aws_iam_policy_document.myecs_task_execution_assume_policy.json
}

data "aws_iam_policy_document" "myecs_task_execution_assume_policy" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
  }
}

resource "aws_iam_role_policy_attachment" "myecs_task_execution_policy" {
  role       = aws_iam_role.myecs_task_execution_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

# task role
resource "aws_iam_role" "myecs_task_role" {
  name               = join("-", [var.base_name, "role"])
  assume_role_policy = data.aws_iam_policy_document.myecs_task_assume_policy.json
}

data "aws_iam_policy_document" "myecs_task_assume_policy" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
  }
}

resource "aws_iam_policy" "myecs_task_policy" {
  name   = join("-", [var.base_name, "policy"])
  policy = data.aws_iam_policy_document.myecs_task_policy.json
}

data "aws_iam_policy_document" "myecs_task_policy" {
  statement {
    actions = [
      "logs:CreateLogStream",
      "logs:CreateLogGroup",
      "logs:DescribeLogStreams",
      "logs:PutLogEvents",
    ]
    effect    = "Allow"
    resources = ["*"]
  }
}

resource "aws_iam_role_policy_attachment" "myecs_task_role" {
  role       = aws_iam_role.myecs_task_role.name
  policy_arn = aws_iam_policy.myecs_task_policy.arn
}
