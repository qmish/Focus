import type { Message } from '../store/roomsStore'

export function buildWebSocketURL(pageURL: string, token: string): string {
  const current = new URL(pageURL)
  const protocol = current.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsURL = new URL(`${protocol}//${current.host}/api/v1/ws`)
  wsURL.searchParams.set('access_token', token)
  return wsURL.toString()
}

export function mergeMessageList(messages: Message[], incoming: Message): Message[] {
  if (messages.some((msg) => msg.id === incoming.id)) {
    return messages
  }
  return [...messages, incoming]
}
