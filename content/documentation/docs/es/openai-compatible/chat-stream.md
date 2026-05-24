Chat Completions con streaming devuelve eventos Server-Sent Events mientras el modelo genera la respuesta. Usa este modo para interfaces de chat, salida en terminal en tiempo real, respuestas largas y escenarios donde importa ver el primer token rapidamente.

Endpoint: `POST /v1/chat/completions`

> [!TIP]
> El probador API al final de esta pagina ya lee respuestas con streaming. Define `"stream": true` en el cuerpo y los fragmentos apareceran en el panel de respuesta al llegar.

## Cuerpo de la solicitud

| Parametro | Tipo | Requerido | Descripcion |
| --- | --- | --- | --- |
| model | string | Si | Nombre del modelo, por ejemplo `gpt-5.4-mini`. |
| messages | array | Si | Mensajes de conversacion en orden, con la misma estructura que el modo sin streaming. |
| stream | boolean | Si | Debe ser `true` para activar streaming. |
| stream_options | object | No | Opciones adicionales de streaming. |
| stream_options.include_usage | boolean | No | Si es `true`, upstreams compatibles pueden enviar uso cerca del final. |
| temperature | number | No | Controla la aleatoriedad. |
| top_p | number | No | Muestreo nucleus. |
| max_tokens | integer | No | Maximo de tokens generados. El ejemplo usa `4096`. |
| tools | array | No | Definiciones de herramientas. Las llamadas a herramientas se devuelven por `delta.tool_calls`. |
| tool_choice | string/object | No | Controla la seleccion de herramientas. |
| response_format | object | No | Solicita un formato de salida. Para JSON, indicalo tambien en el prompt. |

## Formato SSE

| Campo | Descripcion |
| --- | --- |
| data | Linea de datos SSE. Normalmente contiene JSON o `[DONE]`. |
| choices[].delta.content | Texto nuevo de este fragmento. Debe anexarse a la respuesta actual. |
| choices[].delta.role | Rol que puede aparecer al inicio del stream. |
| choices[].delta.tool_calls | Fragmentos incrementales de llamadas a herramientas. Fusiona por `index`. |
| choices[].finish_reason | Motivo de finalizacion. Si no esta vacio, ese choice termino. |
| usage | Puede aparecer al final cuando `stream_options.include_usage=true` y el upstream lo admite. |

## Ejemplo

```bash
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4-mini",
    "stream": true,
    "stream_options": { "include_usage": true },
    "messages": [
      { "role": "user", "content": "Give me three product name ideas." }
    ],
    "max_tokens": 4096
  }'
```

## Stream de ejemplo

```text
data: {"choices":[{"delta":{"role":"assistant"},"index":0}]}

data: {"choices":[{"delta":{"content":"Primera"},"index":0}]}

data: {"choices":[{"delta":{"content":" idea"},"index":0}]}

data: [DONE]
```

## Notas de cliente

- Lee cada linea `data:` y anexa `delta.content` a la respuesta actual.
- Cierra el lector cuando recibas `data: [DONE]`.
- No reintentes indefinidamente la misma generacion despues de una interrupcion de red; podrias duplicar generacion y cobro.
- Si termina con `finish_reason: length`, aumenta `max_tokens` o reduce el prompt.

## Documentacion oficial

- [OpenAI Chat Completions API](https://platform.openai.com/docs/api-reference/chat/create)
- [OpenAI Streaming guide](https://platform.openai.com/docs/guides/streaming-responses)
