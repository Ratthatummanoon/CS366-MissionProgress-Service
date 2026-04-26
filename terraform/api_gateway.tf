# ---------------------------------------------------
# REST API
# ---------------------------------------------------
resource "aws_api_gateway_rest_api" "api" {
  name        = "${var.project_name}-api"
  description = "MissionProgress Service API"

  endpoint_configuration {
    types = ["REGIONAL"]
  }
}

# ---------------------------------------------------
# Lambda Authorizer
# ---------------------------------------------------
resource "aws_api_gateway_authorizer" "lambda_auth" {
  name                             = "${var.project_name}-authorizer"
  rest_api_id                      = aws_api_gateway_rest_api.api.id
  authorizer_uri                   = aws_lambda_function.authorizer.invoke_arn
  authorizer_credentials           = local.lab_role_arn
  type                             = "REQUEST"
  identity_source                  = "method.request.header.x-api-key,method.request.header.X-Rescue-Team-ID"
  authorizer_result_ttl_in_seconds = 300
}

# ---------------------------------------------------
# Resource: /missions
# ---------------------------------------------------
resource "aws_api_gateway_resource" "missions" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_rest_api.api.root_resource_id
  path_part   = "missions"
}

# ---------------------------------------------------
# Resource: /missions/{request_id}
# ---------------------------------------------------
resource "aws_api_gateway_resource" "request_id" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_resource.missions.id
  path_part   = "{request_id}"
}

# ---------------------------------------------------
# Resource: /missions/{request_id}/progress
# ---------------------------------------------------
resource "aws_api_gateway_resource" "mission_progress" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_resource.request_id.id
  path_part   = "progress"
}

# ---------------------------------------------------
# GET /missions/{request_id}
# ---------------------------------------------------
resource "aws_api_gateway_method" "get_mission" {
  rest_api_id   = aws_api_gateway_rest_api.api.id
  resource_id   = aws_api_gateway_resource.request_id.id
  http_method   = "GET"
  authorization = "CUSTOM"
  authorizer_id = aws_api_gateway_authorizer.lambda_auth.id

  request_parameters = {
    "method.request.header.x-api-key"          = true
    "method.request.header.X-Rescue-Team-ID"   = true
  }
}

resource "aws_api_gateway_integration" "get_mission" {
  rest_api_id             = aws_api_gateway_rest_api.api.id
  resource_id             = aws_api_gateway_resource.request_id.id
  http_method             = aws_api_gateway_method.get_mission.http_method
  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = aws_lambda_function.get_mission.invoke_arn
}

# ---------------------------------------------------
# POST /missions/{request_id}/progress
# ---------------------------------------------------
resource "aws_api_gateway_method" "report_progress" {
  rest_api_id   = aws_api_gateway_rest_api.api.id
  resource_id   = aws_api_gateway_resource.mission_progress.id
  http_method   = "POST"
  authorization = "CUSTOM"
  authorizer_id = aws_api_gateway_authorizer.lambda_auth.id

  request_parameters = {
    "method.request.header.x-api-key"          = true
    "method.request.header.X-Rescue-Team-ID"   = true
  }
}

resource "aws_api_gateway_integration" "report_progress" {
  rest_api_id             = aws_api_gateway_rest_api.api.id
  resource_id             = aws_api_gateway_resource.mission_progress.id
  http_method             = aws_api_gateway_method.report_progress.http_method
  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = aws_lambda_function.report_progress.invoke_arn
}

# ---------------------------------------------------
# OPTIONS /missions/{request_id} (CORS)
# ---------------------------------------------------
resource "aws_api_gateway_method" "options_incident" {
  rest_api_id   = aws_api_gateway_rest_api.api.id
  resource_id   = aws_api_gateway_resource.request_id.id
  http_method   = "OPTIONS"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "options_incident" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  resource_id = aws_api_gateway_resource.request_id.id
  http_method = aws_api_gateway_method.options_incident.http_method
  type        = "MOCK"

  request_templates = {
    "application/json" = "{\"statusCode\": 200}"
  }
}

resource "aws_api_gateway_method_response" "options_incident" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  resource_id = aws_api_gateway_resource.request_id.id
  http_method = aws_api_gateway_method.options_incident.http_method
  status_code = "200"

  response_parameters = {
    "method.response.header.Access-Control-Allow-Headers" = true
    "method.response.header.Access-Control-Allow-Methods" = true
    "method.response.header.Access-Control-Allow-Origin"  = true
  }

  response_models = {
    "application/json" = "Empty"
  }
}

resource "aws_api_gateway_integration_response" "options_incident" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  resource_id = aws_api_gateway_resource.request_id.id
  http_method = aws_api_gateway_method.options_incident.http_method
  status_code = aws_api_gateway_method_response.options_incident.status_code

  response_parameters = {
    "method.response.header.Access-Control-Allow-Headers" = "'Content-Type,x-api-key,X-Rescue-Team-ID'"
    "method.response.header.Access-Control-Allow-Methods" = "'GET,POST,OPTIONS'"
    "method.response.header.Access-Control-Allow-Origin"  = "'*'"
  }
}

# ---------------------------------------------------
# OPTIONS /missions/{request_id}/progress (CORS)
# ---------------------------------------------------
resource "aws_api_gateway_method" "options_progress" {
  rest_api_id   = aws_api_gateway_rest_api.api.id
  resource_id   = aws_api_gateway_resource.mission_progress.id
  http_method   = "OPTIONS"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "options_progress" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  resource_id = aws_api_gateway_resource.mission_progress.id
  http_method = aws_api_gateway_method.options_progress.http_method
  type        = "MOCK"

  request_templates = {
    "application/json" = "{\"statusCode\": 200}"
  }
}

resource "aws_api_gateway_method_response" "options_progress" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  resource_id = aws_api_gateway_resource.mission_progress.id
  http_method = aws_api_gateway_method.options_progress.http_method
  status_code = "200"

  response_parameters = {
    "method.response.header.Access-Control-Allow-Headers" = true
    "method.response.header.Access-Control-Allow-Methods" = true
    "method.response.header.Access-Control-Allow-Origin"  = true
  }

  response_models = {
    "application/json" = "Empty"
  }
}

resource "aws_api_gateway_integration_response" "options_progress" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  resource_id = aws_api_gateway_resource.mission_progress.id
  http_method = aws_api_gateway_method.options_progress.http_method
  status_code = aws_api_gateway_method_response.options_progress.status_code

  response_parameters = {
    "method.response.header.Access-Control-Allow-Headers" = "'Content-Type,x-api-key,X-Rescue-Team-ID'"
    "method.response.header.Access-Control-Allow-Methods" = "'GET,POST,OPTIONS'"
    "method.response.header.Access-Control-Allow-Origin"  = "'*'"
  }
}

# ---------------------------------------------------
# Resource: /missions/{request_id}/presigned-url
# ---------------------------------------------------
resource "aws_api_gateway_resource" "mission_presigned_url" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_resource.request_id.id
  path_part   = "presigned-url"
}

# ---------------------------------------------------
# POST /missions/{request_id}/presigned-url
# ---------------------------------------------------
resource "aws_api_gateway_method" "presigned_url" {
  rest_api_id   = aws_api_gateway_rest_api.api.id
  resource_id   = aws_api_gateway_resource.mission_presigned_url.id
  http_method   = "POST"
  authorization = "CUSTOM"
  authorizer_id = aws_api_gateway_authorizer.lambda_auth.id

  request_parameters = {
    "method.request.header.x-api-key"        = true
    "method.request.header.X-Rescue-Team-ID" = true
  }
}

resource "aws_api_gateway_integration" "presigned_url" {
  rest_api_id             = aws_api_gateway_rest_api.api.id
  resource_id             = aws_api_gateway_resource.mission_presigned_url.id
  http_method             = aws_api_gateway_method.presigned_url.http_method
  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = aws_lambda_function.presigned_url.invoke_arn
}

# ---------------------------------------------------
# OPTIONS /missions/{request_id}/presigned-url (CORS)
# ---------------------------------------------------
resource "aws_api_gateway_method" "options_presigned_url" {
  rest_api_id   = aws_api_gateway_rest_api.api.id
  resource_id   = aws_api_gateway_resource.mission_presigned_url.id
  http_method   = "OPTIONS"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "options_presigned_url" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  resource_id = aws_api_gateway_resource.mission_presigned_url.id
  http_method = aws_api_gateway_method.options_presigned_url.http_method
  type        = "MOCK"

  request_templates = {
    "application/json" = "{\"statusCode\": 200}"
  }
}

resource "aws_api_gateway_method_response" "options_presigned_url" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  resource_id = aws_api_gateway_resource.mission_presigned_url.id
  http_method = aws_api_gateway_method.options_presigned_url.http_method
  status_code = "200"

  response_parameters = {
    "method.response.header.Access-Control-Allow-Headers" = true
    "method.response.header.Access-Control-Allow-Methods" = true
    "method.response.header.Access-Control-Allow-Origin"  = true
  }

  response_models = {
    "application/json" = "Empty"
  }
}

resource "aws_api_gateway_integration_response" "options_presigned_url" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  resource_id = aws_api_gateway_resource.mission_presigned_url.id
  http_method = aws_api_gateway_method.options_presigned_url.http_method
  status_code = aws_api_gateway_method_response.options_presigned_url.status_code

  response_parameters = {
    "method.response.header.Access-Control-Allow-Headers" = "'Content-Type,x-api-key,X-Rescue-Team-ID'"
    "method.response.header.Access-Control-Allow-Methods" = "'POST,OPTIONS'"
    "method.response.header.Access-Control-Allow-Origin"  = "'*'"
  }
}

# ---------------------------------------------------
# GET /missions (list-missions)
# ---------------------------------------------------
resource "aws_api_gateway_method" "list_missions" {
  rest_api_id   = aws_api_gateway_rest_api.api.id
  resource_id   = aws_api_gateway_resource.missions.id
  http_method   = "GET"
  authorization = "CUSTOM"
  authorizer_id = aws_api_gateway_authorizer.lambda_auth.id

  request_parameters = {
    "method.request.header.x-api-key"        = true
    "method.request.header.X-Rescue-Team-ID" = true
  }
}

resource "aws_api_gateway_integration" "list_missions" {
  rest_api_id             = aws_api_gateway_rest_api.api.id
  resource_id             = aws_api_gateway_resource.missions.id
  http_method             = aws_api_gateway_method.list_missions.http_method
  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = aws_lambda_function.list_missions.invoke_arn
}

# ---------------------------------------------------
# OPTIONS /missions (CORS)
# ---------------------------------------------------
resource "aws_api_gateway_method" "options_incidents" {
  rest_api_id   = aws_api_gateway_rest_api.api.id
  resource_id   = aws_api_gateway_resource.missions.id
  http_method   = "OPTIONS"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "options_incidents" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  resource_id = aws_api_gateway_resource.missions.id
  http_method = aws_api_gateway_method.options_incidents.http_method
  type        = "MOCK"

  request_templates = {
    "application/json" = "{\"statusCode\": 200}"
  }
}

resource "aws_api_gateway_method_response" "options_incidents" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  resource_id = aws_api_gateway_resource.missions.id
  http_method = aws_api_gateway_method.options_incidents.http_method
  status_code = "200"

  response_parameters = {
    "method.response.header.Access-Control-Allow-Headers" = true
    "method.response.header.Access-Control-Allow-Methods" = true
    "method.response.header.Access-Control-Allow-Origin"  = true
  }

  response_models = {
    "application/json" = "Empty"
  }
}

resource "aws_api_gateway_integration_response" "options_incidents" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  resource_id = aws_api_gateway_resource.missions.id
  http_method = aws_api_gateway_method.options_incidents.http_method
  status_code = aws_api_gateway_method_response.options_incidents.status_code

  response_parameters = {
    "method.response.header.Access-Control-Allow-Headers" = "'Content-Type,x-api-key,X-Rescue-Team-ID'"
    "method.response.header.Access-Control-Allow-Methods" = "'GET,OPTIONS'"
    "method.response.header.Access-Control-Allow-Origin"  = "'*'"
  }
}

# ---------------------------------------------------
# Deployment & Stage
# ---------------------------------------------------
resource "aws_api_gateway_deployment" "deployment" {
  rest_api_id = aws_api_gateway_rest_api.api.id

  triggers = {
    redeployment = sha1(jsonencode([
      aws_api_gateway_resource.missions.id,
      aws_api_gateway_resource.request_id.id,
      aws_api_gateway_resource.mission_progress.id,
      aws_api_gateway_resource.mission_presigned_url.id,
      aws_api_gateway_method.get_mission.id,
      aws_api_gateway_integration.get_mission.id,
      aws_api_gateway_method.report_progress.id,
      aws_api_gateway_integration.report_progress.id,
      aws_api_gateway_method.presigned_url.id,
      aws_api_gateway_integration.presigned_url.id,
      aws_api_gateway_method.list_missions.id,
      aws_api_gateway_integration.list_missions.id,
      aws_api_gateway_method.options_incident.id,
      aws_api_gateway_method.options_progress.id,
      aws_api_gateway_method.options_presigned_url.id,
      aws_api_gateway_method.options_incidents.id,
      aws_api_gateway_authorizer.lambda_auth.id,
    ]))
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_api_gateway_stage" "v1" {
  deployment_id = aws_api_gateway_deployment.deployment.id
  rest_api_id   = aws_api_gateway_rest_api.api.id
  stage_name    = "v1"

  tags = {
    Project = var.project_name
  }
}
