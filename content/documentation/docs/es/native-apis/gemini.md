La API nativa de Gemini usa el formato `generateContent` de Google Gemini. Es util si tu cliente ya usa SDKs de Gemini o si necesitas campos nativos como `contents`, `generationConfig` y `safetySettings`. Para compatibilidad amplia con clientes, usa preferentemente Chat Completions compatible con OpenAI.

Endpoint: `POST /v1beta/models/{model}:generateContent`

> [!NOTE]
> `{model}` forma parte de la ruta, por ejemplo `gemini-2.5-flash`. Los modelos y precios disponibles dependen de la configuracion de la consola.

## Encabezados

| Parametro | Tipo | Requerido | Descripcion |
| --- | --- | --- | --- |
| Authorization | string | Si | Usa `Bearer YOUR_API_KEY`. |
| Content-Type | string | Si | Debe ser `application/json`. |

## Parametros de ruta

| Parametro | Tipo | Requerido | Descripcion |
| --- | --- | --- | --- |
| model | string | Si | Nombre del modelo Gemini, por ejemplo `gemini-2.5-flash`. El probador sincroniza el campo de modelo con el segmento `{model}` de la URL. |

## Cuerpo de la solicitud

| Parametro | Tipo | Requerido | Descripcion |
| --- | --- | --- | --- |
| contents | array | Si | Array de contenido de entrada. Agrega varios elementos en orden para conversaciones multi-turno. |
| contents[].role | string | No | Rol, normalmente `user` o `model`. En una solicitud de texto simple suele usarse `user`. |
| contents[].parts | array | Si | Partes del contenido. Texto, imagenes y archivos se representan como parts. |
| contents[].parts[].text | string | No | Entrada de texto. |
| systemInstruction | object | No | Instruccion de sistema que guia el comportamiento del modelo. |
| generationConfig | object | No | Configuracion de generacion. |
| generationConfig.temperature | number | No | Controla la aleatoriedad. |
| generationConfig.topP | number | No | Muestreo nucleus. |
| generationConfig.topK | integer | No | Muestrea entre los K tokens mas probables. |
| generationConfig.maxOutputTokens | integer | No | Maximo de tokens de salida. El ejemplo usa `4096`. |
| generationConfig.stopSequences | array | No | Detiene la generacion cuando aparece alguna secuencia. |
| safetySettings | array | No | Politicas de seguridad, segun soporte del upstream. |
| tools | array | No | Definiciones nativas de herramientas Gemini, como function calling. |
| toolConfig | object | No | Configuracion de llamadas a herramientas. |

## Ejemplo

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

## Campos de respuesta

| Campo | Descripcion |
| --- | --- |
| candidates | Array de respuestas candidatas. Normalmente se lee la primera. |
| candidates[].content.parts[].text | Texto generado. |
| candidates[].finishReason | Motivo de finalizacion, como `STOP`, `MAX_TOKENS` o `SAFETY`. |
| candidates[].safetyRatings | Detalles de evaluacion de seguridad. |
| usageMetadata.promptTokenCount | Tokens de entrada. |
| usageMetadata.candidatesTokenCount | Tokens de salida. |
| usageMetadata.totalTokenCount | Tokens totales. |

## Documentacion oficial

- [Gemini generateContent API](https://ai.google.dev/api/generate-content)
- [Gemini function calling](https://ai.google.dev/gemini-api/docs/function-calling)
