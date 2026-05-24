The Gemini native API uses the Google Gemini `generateContent` request shape. It is useful when your client already uses Gemini SDKs or when you need native fields such as `contents`, `generationConfig`, and `safetySettings`. For broad client compatibility, prefer the OpenAI-compatible Chat Completions API.

Endpoint: `POST /v1beta/models/{model}:generateContent`

> [!NOTE]
> `{model}` is part of the URL path, for example `gemini-2.5-flash`. Available models and prices depend on your console configuration.

## Headers

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| Authorization | string | Yes | Use `Bearer YOUR_API_KEY`. |
| Content-Type | string | Yes | Must be `application/json`. |

## Path Parameters

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| model | string | Yes | Gemini model name, for example `gemini-2.5-flash`. The tester syncs the model input into the `{model}` URL segment. |

## Request Body

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| contents | array | Yes | Input content array. Add multiple items in order for multi-turn conversations. |
| contents[].role | string | No | Role value, usually `user` or `model`. Single-turn text requests commonly use `user`. |
| contents[].parts | array | Yes | Content parts. Text, images, and files are represented as parts. |
| contents[].parts[].text | string | No | Text input. |
| systemInstruction | object | No | System-level instruction that guides model behavior. |
| generationConfig | object | No | Generation settings. |
| generationConfig.temperature | number | No | Controls randomness. |
| generationConfig.topP | number | No | Nucleus sampling value. |
| generationConfig.topK | integer | No | Samples from the top K likely tokens. |
| generationConfig.maxOutputTokens | integer | No | Maximum output tokens. The example uses `4096`. |
| generationConfig.stopSequences | array | No | Stop generation when any sequence appears. |
| safetySettings | array | No | Safety policy settings, depending on upstream support. |
| tools | array | No | Gemini native tool definitions, such as function calling. |
| toolConfig | object | No | Tool-calling configuration. |

## Example Request

```bash
curl https://aivrae.com/v1beta/models/gemini-2.5-flash:generateContent \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [
      {
        "role": "user",
        "parts": [
          { "text": "Write a concise project summary." }
        ]
      }
    ],
    "generationConfig": {
      "temperature": 0.7,
      "maxOutputTokens": 4096
    }
  }'
```

## Response Fields

| Field | Description |
| --- | --- |
| candidates | Candidate response array. Most clients read the first candidate. |
| candidates[].content.parts[].text | Generated text. |
| candidates[].finishReason | Stop reason, such as `STOP`, `MAX_TOKENS`, or `SAFETY`. |
| candidates[].safetyRatings | Safety rating details. |
| usageMetadata.promptTokenCount | Input token count. |
| usageMetadata.candidatesTokenCount | Output token count. |
| usageMetadata.totalTokenCount | Total token count. |

## Official Docs

- [Gemini generateContent API](https://ai.google.dev/api/generate-content)
- [Gemini function calling](https://ai.google.dev/gemini-api/docs/function-calling)
