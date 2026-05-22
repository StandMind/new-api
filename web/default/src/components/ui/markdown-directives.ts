/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import type { Node } from 'unist'
import { visit } from 'unist-util-visit'

type DirectiveNode = Node & {
  name?: string
  attributes?: Record<string, boolean | number | string | null | undefined>
  children?: DirectiveNode[]
  data?: {
    hName?: string
    hProperties?: Record<string, string>
  }
  value?: string
}

const directiveElementNames: Record<string, string> = {
  api: 'api-section',
  endpoint: 'api-endpoint',
  parameters: 'api-parameters',
  params: 'api-parameters',
  request: 'api-request',
  response: 'api-response',
  section: 'api-section',
  tab: 'api-tab',
  tabs: 'api-tabs',
}

const directiveNodeTypes = new Set([
  'containerDirective',
  'leafDirective',
  'textDirective',
])

const httpMethods = new Set([
  'DELETE',
  'GET',
  'HEAD',
  'OPTIONS',
  'PATCH',
  'POST',
  'PUT',
])

function nodeText(node: DirectiveNode | undefined): string {
  if (!node) return ''
  if (typeof node.value === 'string') return node.value
  return node.children?.map(nodeText).join('') ?? ''
}

function directiveLabel(node: DirectiveNode) {
  return node.children
    ?.filter((child) => child.type === 'text')
    .map(nodeText)
    .join('')
    .trim()
}

function directiveAttributes(node: DirectiveNode) {
  const properties: Record<string, string> = {}

  Object.entries(node.attributes ?? {}).forEach(([key, value]) => {
    if (value === null || value === undefined || value === false) return
    properties[key] = value === true ? 'true' : String(value)
  })

  return properties
}

function applyEndpointDefaults(
  properties: Record<string, string>,
  label?: string
) {
  if (!label) return

  const [methodCandidate, ...pathParts] = label.split(/\s+/)
  const method = methodCandidate?.toUpperCase()
  if (method && httpMethods.has(method) && pathParts.length > 0) {
    properties.method ??= method
    properties.path ??= pathParts.join(' ')
    return
  }

  properties.title ??= label
}

function applyDirectiveDefaults(
  node: DirectiveNode,
  properties: Record<string, string>
) {
  const label = directiveLabel(node)

  switch (node.name) {
    case 'endpoint':
      applyEndpointDefaults(properties, label)
      break
    case 'tab':
      if (label) properties.title ??= label
      break
    case 'request':
      properties.title ??= label || 'Request'
      break
    case 'response':
      properties.title ??= label || 'Response'
      break
    case 'parameters':
    case 'params':
      properties.title ??= label || 'Parameters'
      break
    default:
      if (label) properties.title ??= label
      break
  }
}

export function remarkApiDirectives() {
  return (tree: Node) => {
    visit(tree, (node: Node) => {
      if (!directiveNodeTypes.has(node.type)) return

      const directiveNode = node as DirectiveNode
      const elementName = directiveNode.name
        ? directiveElementNames[directiveNode.name]
        : undefined
      if (!elementName) return

      const properties = directiveAttributes(directiveNode)
      applyDirectiveDefaults(directiveNode, properties)

      directiveNode.data ??= {}
      directiveNode.data.hName = elementName
      directiveNode.data.hProperties = properties
    })
  }
}
