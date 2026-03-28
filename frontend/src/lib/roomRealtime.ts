import type { Message } from '../store/roomsStore'

export function buildWebSocketURL(pageURL: string, _token?: string): string {
  const current = new URL(pageURL)
  const protocol = current.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${current.host}/api/v1/ws`
}

export function mergeMessageList(messages: Message[], incoming: Message): Message[] {
  if (messages.some((msg) => msg.id === incoming.id)) {
    return messages
  }
  return [...messages, incoming]
}
