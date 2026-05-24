Chat Completions sin streaming devuelve una respuesta JSON completa cuando termina la generacion. Usa este modo para textos cortos, clasificacion, resumenes, extraccion estructurada y tareas en segundo plano que no necesitan mostrar tokens en tiempo real.

Endpoint: `POST /v1/chat/completions`

> [!NOTE]
> El formato sigue el estilo de OpenAI Chat Completions. Los modelos disponibles, parametros admitidos y facturacion dependen de los modelos y canales upstream habilitados en la consola.

## Encabezados

| Parametro | Tipo | Requerido | Descripcion |
| --- | --- | --- | --- |
| Authorization | string | Si | Usa `Bearer YOUR_API_KEY`. |
| Content-Type | string | Si | Debe ser `application/json`. |

## Cuerpo de la solicitud

| Parametro | Tipo | Requerido | Descripcion |
| --- | --- | --- | --- |
| model | string | Si | Nombre del modelo, por ejemplo `gpt-5.4-mini`. Usa un modelo disponible en la lista de modelos o consola. |
| messages | array | Si | Mensajes de conversacion en orden. |
| messages[].role | string | Si | Rol del mensaje. Valores comunes: `system`, `user`, `assistant`, `tool`. |
| messages[].content | string/array | Si | Contenido del mensaje. Usa string para texto o array para contenido multimodal en modelos compatibles. |
| temperature | number | No | Aleatoriedad de muestreo, normalmente de `0` a `2`. |
| top_p | number | No | Muestreo nucleus. Evita ajustar mucho `temperature` y `top_p` al mismo tiempo. |
| max_tokens | integer | No | Maximo de tokens generados. El ejemplo usa `4096`; el limite real depende del contexto del modelo y del upstream. |
| stream | boolean | No | Omite este campo o usa `false` para solicitudes sin streaming. |
| stop | string/array | No | Detiene la generacion cuando aparece alguna secuencia indicada. |
| tools | array | No | Definiciones de funciones o herramientas, segun soporte del modelo. |
| tool_choice | string/object | No | Controla la seleccion de herramientas, como `auto`, `none` o una herramienta concreta. |
| response_format | object | No | Solicita un formato de salida concreto, como objeto JSON o JSON Schema. |
| presence_penalty | number | No | Penaliza temas repetidos. Normalmente entre `-2` y `2`. |
| frequency_penalty | number | No | Penaliza palabras repetidas. Normalmente entre `-2` y `2`. |
| user | string | No | Identificador del usuario final para auditoria y control de riesgo. |

## Ejemplo

```bash
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4-mini",
    "messages": [
      { "role": "system", "content": "You are a concise assistant." },
      { "role": "user", "content": "Write a short welcome message." }
    ],
    "temperature": 0.7,
    "max_tokens": 4096
  }'
```

## Campos de respuesta

| Campo | Descripcion |
| --- | --- |
| id | Identificador de la respuesta. |
| object | Tipo de objeto, normalmente `chat.completion`. |
| created | Marca de tiempo Unix de creacion. |
| model | Modelo que genero la respuesta. |
| choices[].message.role | Rol devuelto, normalmente `assistant`. |
| choices[].message.content | Texto principal generado. |
| choices[].finish_reason | Motivo de finalizacion, como `stop`, `length` o `tool_calls`. `length` indica que se alcanzo el limite de salida. |
| usage.prompt_tokens | Tokens de entrada. |
| usage.completion_tokens | Tokens de salida. |
| usage.total_tokens | Tokens totales, util para revisar facturacion. |

## Documentacion oficial

- [OpenAI Chat Completions API](https://platform.openai.com/docs/api-reference/chat/create)
- [OpenAI Text generation guide](https://platform.openai.com/docs/guides/text-generation)
