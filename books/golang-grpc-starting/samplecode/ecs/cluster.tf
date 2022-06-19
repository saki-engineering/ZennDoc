resource "aws_ecs_cluster" "myecs" {
  name = join("-", [var.base_name, "cluster"])
}

resource "aws_ecs_cluster_capacity_providers" "myecs" {
  cluster_name = aws_ecs_cluster.myecs.name

  capacity_providers = ["FARGATE"]

  default_capacity_provider_strategy {
    base              = 1
    weight            = 100
    capacity_provider = "FARGATE"
  }
}