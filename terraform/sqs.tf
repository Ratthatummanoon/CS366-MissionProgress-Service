# ---------------------------------------------------
# SQS Dead Letter Queue for mission-dispatch
# ---------------------------------------------------
resource "aws_sqs_queue" "mission_dispatch_dlq" {
  name                      = "${var.project_name}-dispatch-dlq"
  message_retention_seconds = 1209600 # 14 days

  tags = {
    Project = var.project_name
  }
}

# ---------------------------------------------------
# SQS Queue: mission-dispatch (SNS → SQS → Lambda)
#
# ทำหน้าที่ buffer message จาก ManageDispatch SNS topic
# ก่อนให้ mission-assigned-handler Lambda ดึงไปประมวลผลทีละ 1 message
# เพื่อควบคุม Lambda concurrency ไม่ให้ชน Learner Lab limit (10)
# ---------------------------------------------------
resource "aws_sqs_queue" "mission_dispatch_queue" {
  name                       = "${var.project_name}-dispatch-queue"
  visibility_timeout_seconds = 30  # ต้องมากกว่าหรือเท่ากับ Lambda timeout (20s)
  message_retention_seconds  = 86400 # 1 day

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.mission_dispatch_dlq.arn
    maxReceiveCount     = 3 # ลอง retry 3 ครั้ง แล้วส่งไป DLQ
  })

  tags = {
    Project = var.project_name
  }
}

# ---------------------------------------------------
# SQS Queue Policy: อนุญาตให้ SNS ส่ง message เข้า Queue ได้
# ---------------------------------------------------
resource "aws_sqs_queue_policy" "mission_dispatch_queue" {
  count     = var.dispatch_sns_topic_arn != "" ? 1 : 0
  queue_url = aws_sqs_queue.mission_dispatch_queue.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "AllowSNSPublish"
        Effect    = "Allow"
        Principal = { Service = "sns.amazonaws.com" }
        Action    = "sqs:SendMessage"
        Resource  = aws_sqs_queue.mission_dispatch_queue.arn
        Condition = {
          ArnEquals = {
            "aws:SourceArn" = var.dispatch_sns_topic_arn
          }
        }
      }
    ]
  })
}
