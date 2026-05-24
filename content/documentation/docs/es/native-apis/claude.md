La API nativa de Claude usa el protocolo Anthropic Messages. Es util si tu cliente ya usa SDKs de Claude o si necesitas campos nativos como `system`, `tools`, `thinking` y `stream`. Para compatibilidad amplia con clientes, usa preferentemente Chat Completions compatible con OpenAI.

Endpoint: `POST /v1/messages`

> [!NOTE]
> Las solicitudes nativas de Claude usan el encabezado `x-api-key` en vez de `Authorization: Bearer ...`. El probador configura el encabezado de autenticacion correcto para este endpoint.

## Encabezados

| Parametro | Tipo | Requerido | Descripcion |
| --- | --- | --- | --- |
| x-api-key | string | Si | Tu API Key. |
| anthropic-version | string | Si | Version de la API de Anthropic. El ejemplo usa `2023-06-01`. |
| Content-Type | string | Si | Debe ser `application/json`. |

## Cuerpo de la solicitud

| Parametro | Tipo | Requerido | Descripcion |
| --- | --- | --- | --- |
| model | string | Si | Nombre del modelo Claude. Usa un modelo habilitado en la consola. |
| max_tokens | integer | Si | Maximo de tokens de salida. El ejemplo usa `4096`. |
| messages | array | Si | Mensajes de conversacion. |
| messages[].role | string | Si | Rol del mensaje, normalmente `user` o `assistant`. |
| messages[].content | string/array | Si | Contenido del mensaje. Usa string para texto o bloques de contenido para multimodal. |
| system | string/array | No | Prompt de sistema que guia el comportamiento del asistente. |
| temperature | number | No | Controla la aleatoriedad. |
| top_p | number | No | Muestreo nucleus. |
| top_k | integer | No | Limita los tokens candidatos. |
| stop_sequences | array | No | Detiene la generacion cuando aparece alguna secuencia. |
| stream | boolean | No | Usa `true` para streaming nativo de Claude. |
| tools | array | No | Definiciones de herramientas. |
| tool_choice | object | No | Controla la seleccion de herramientas. |
| metadata | object | No | Metadatos opcionales de usuario o negocio. |
| thinking | object | No | Configuracion de pensamiento extendido. Solo aplica en modelos compatibles. |

## Ejemplo

```bash
curl https://aivrae.com/v1/messages \
  -H "x-api-key: YOUR_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "max_tokens": 4096,
    "messages": [
      { "role": "user", "content": "Write a concise project summary." }
    ]
  }'
```

## Campos de respuesta

| Campo | Descripcion |
| --- | --- |
| id | Identificador de respuesta Claude. |
| type | Tipo de objeto, normalmente `message`. |
| role | Rol devuelto, normalmente `assistant`. |
| content | Array de bloques de contenido. El texto suele estar en `content[].text`. |
| model | Modelo que genero la respuesta. |
| stop_reason | Motivo de finalizacion, como `end_turn`, `max_tokens` o `tool_use`. |
| usage.input_tokens | Tokens de entrada. |
| usage.output_tokens | Tokens de salida. |

## Documentacion oficial

- [Anthropic Messages API](https://docs.anthropic.com/en/api/messages)
- [Anthropic streaming Messages](https://docs.anthropic.com/en/api/messages-streaming)
