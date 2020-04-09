resource "aws_iam_role" "usage_lambda_role" {
  name = "usage_lambda_role"

  assume_role_policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
      {
        "Action": "sts:AssumeRole",
        "Principal": {
          "Service": "lambda.amazonaws.com"
        },
        "Effect": "Allow",
        "Sid": ""
      }
    ]
  }
EOF
}

resource "aws_iam_policy" "usage_lambda_policy" {
  name        = "usage_lambda_policy"
  description = "A policy for lambda usage function"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "ec2:Describe*",
        "logs:CreateLogStream",
        "logs:CreateLogGroup",
        "logs:PutLogEvents",
        "cloudwatch:GetMetricData",
        "cloudwatch:GetMetricStatistics",
        "cloudwatch:ListMetrics",
        "elasticloadbalancing:DescribeLoadBalancers",
        "elasticloadbalancing:DescribeTags",
        "cloudfront:ListDistributions",
        "s3:ListAllMyBuckets",
        "s3:ListBucket"
      ],
      "Effect": "Allow",
      "Resource": "*"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy_attachment" "usage-lambda-policy-attachment" {
  role       = aws_iam_role.usage_lambda_role.name
  policy_arn = aws_iam_policy.usage_lambda_policy.arn
}

resource "aws_lambda_function" "usage-lambda" {

  function_name = "${var.env_name}-usage-lambda"
  s3_bucket = var.s3_bucket
  s3_key = "usage_lambda.zip"

  role          =  aws_iam_role.usage_lambda_role.arn
  handler       = "usage_lambda"

  runtime = "go1.x"
  reserved_concurrent_executions = 1
  timeout = 50
  environment {
    variables = {
      anodotUrl = "${var.anodotUrl}"
      token = "${var.token}"
    }
  }
}

resource "aws_cloudwatch_event_rule" "cronjob_rule" {
    name        = "${var.env_name}_cronjob_rule"
    description = "Just cron like shceduler"

    schedule_expression = "rate(20 minutes)"
}

resource "aws_cloudwatch_event_target" "lambda" {
  rule      = aws_cloudwatch_event_rule.cronjob_rule.name
  target_id = "TargetFunction"
  arn       =  aws_lambda_function.usage-lambda.arn
}

resource "aws_lambda_permission" "allow_cloudwatch" {
  statement_id  = "AllowExecutionFromCloudWatch"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.usage-lambda.function_name
  principal     = "events.amazonaws.com"
  source_arn    =  aws_cloudwatch_event_rule.cronjob_rule.arn
}