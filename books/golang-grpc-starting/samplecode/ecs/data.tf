data "aws_ecr_repository" "myecs" {
  name = "your-repo-name"
}

data "aws_vpc" "myecs" {
  // (略)
  // ALBを動かしたいVPCを参照できるように、適切に引数を設定してください
}

data "aws_subnets" "myecs_public" {
  // (略)
  // ALBを動かしたいパブリックサブネットを参照できるように、適切に引数を設定してください
}

data "aws_subnets" "myecs_private" {
  // (略)
  // ALBを動かしたいプライベートサブネットを参照できるように、適切に引数を設定してください
}

data "aws_acm_certificate" "myecs" {
  domain = "yourdomain.example.com"
}