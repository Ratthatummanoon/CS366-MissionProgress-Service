## Q4: Lack of Observability

> **"Is `X-Trace-Id` in every response header? Are error responses structured with `traceId` and `message`?"**

### คำตอบ

**`X-Trace-Id` ในทุก response header?**

```go
// response.go — buildHeaders() เพิ่ม X-Trace-Id ทุกครั้ง
func buildHeaders(traceID string) map[string]string {
    headers["X-Trace-Id"] = traceID  // ✅ มีทุก response
}
```

| Endpoint | มี `X-Trace-Id`? |
|---|:---:|
| `GET /incidents/{id}` (success) | ✅ |
| `GET /incidents/{id}` (error) | ✅ |
| `POST /incidents/{id}/progress` (success) | ✅ |
| `POST /incidents/{id}/progress` (error) | ✅ |

**Error response มี `traceId` + `message`?**

```json
{
    "error": "INVALID_STATE_TRANSITION",
    "code": "INVALID_STATE_TRANSITION",
    "message": "Cannot transition from EN_ROUTE to RESOLVED",
    "traceId": "7c9e6679-7425-40de-944b-e07fc1f90ae7"
}
```

| เกณฑ์ | มี? |
|---|:---:|
| `message` ใน error body | ✅ |
| `traceId` ใน error body | ✅ |
| `X-Trace-Id` ใน header ตรงกับ `traceId` ใน body | ✅ |

✅ สรุป: **ผ่าน**  `X-Trace-Id` อยู่ในทุก response header และ error response มีทั้ง `traceId` และ `message`

---

