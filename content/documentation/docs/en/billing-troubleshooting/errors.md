Applications should handle common status codes explicitly and log service error messages on the backend.

| Status | Meaning | Suggested handling |
| --- | --- | --- |
| 400 Bad Request | Invalid body, unsupported parameter, or malformed JSON | Check body, endpoint compatibility, and parameter names. |
| 401 Unauthorized | Missing, wrong, expired, or disabled key | Copy the key again and verify header format. |
| 403 Forbidden | No permission for model, group, or feature | Check model permissions, plan, and routing group. |
| 404 Not Found | Wrong path or model not found | Verify Base URL, native endpoint path, and model name. |
| 413 Request Entity Too Large | Request body or context too large | Reduce messages, contents, or input size. |
| 429 Too Many Requests | Rate, concurrency, quota, or balance limit | Reduce concurrency, check balance, use backoff retry. |
| 500 / 502 / 503 | Temporary Aivrae or model service failure | Retry later; contact support if persistent. |
