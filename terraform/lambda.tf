# ---------------------------------------------------
# Lambda: report-progress
# ---------------------------------------------------
resource "aws_lambda_function" "report_progress" {
  function_name = "${var.project_name}-report-progress"
  role          = local.lab_role_arn
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  memory_size   = 256
  timeout       = 30
  filename      = "${path.module}/build/report-progress.zip"

  source_code_hash = fileexists("${path.module}/build/report-progress.zip") ? filebase64sha256("${path.module}/build/report-progress.zip") : null

  environment {
    variables = {
      TABLE_MISSION        = aws_dynamodb_table.mission_assignment.name
      TABLE_TIMELINE       = aws_dynamodb_table.mission_timeline.name
      TABLE_OUTBOX         = aws_dynamodb_table.event_outbox.name
      EVENT_BUS_NAME       = aws_cloudwatch_event_bus.mission_events.name
      INCIDENT_SERVICE_URL = var.incident_service_url
    }
  }

  tags = {
    Project = var.project_name
  }
}

# ---------------------------------------------------
# Lambda: get-mission
# ---------------------------------------------------
resource "aws_lambda_function" "get_mission" {
  function_name = "${var.project_name}-get-mission"
  role          = local.lab_role_arn
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  memory_size   = 256
  timeout       = 30
  filename      = "${path.module}/build/get-mission.zip"

  source_code_hash = fileexists("${path.module}/build/get-mission.zip") ? filebase64sha256("${path.module}/build/get-mission.zip") : null

  environment {
    variables = {
      TABLE_MISSION        = aws_dynamodb_table.mission_assignment.name
      TABLE_TIMELINE       = aws_dynamodb_table.mission_timeline.name
      INCIDENT_SERVICE_URL = var.incident_service_url
    }
  }

  tags = {
    Project = var.project_name
  }
}

# ---------------------------------------------------
# Lambda: authorizer
# ---------------------------------------------------
resource "aws_lambda_function" "authorizer" {
  function_name = "${var.project_name}-authorizer"
  role          = local.lab_role_arn
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  memory_size   = 256
  timeout       = 30
  filename      = "${path.module}/build/authorizer.zip"

  source_code_hash = fileexists("${path.module}/build/authorizer.zip") ? filebase64sha256("${path.module}/build/authorizer.zip") : null

  environment {
    variables = {
      VALID_API_KEY = var.api_key_value
    }
  }

  tags = {
    Project = var.project_name
  }
}

# ---------------------------------------------------
# Lambda Permissions for API Gateway
# ---------------------------------------------------
resource "aws_lambda_permission" "apigw_report_progress" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.report_progress.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_rest_api.api.execution_arn}/*/*"
}

resource "aws_lambda_permission" "apigw_get_mission" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.get_mission.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_rest_api.api.execution_arn}/*/*"
}

resource "aws_lambda_permission" "apigw_authorizer" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.authorizer.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_rest_api.api.execution_arn}/*/*"
}
