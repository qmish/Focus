export function isDeliveryFailed(success: boolean): boolean {
  return !success
}

export function botStatusLabel(status: string): string {
  switch (status) {
    case 'failed':
      return 'Ошибка'
    case 'permission_denied':
      return 'Нет доступа'
    case 'rate_limited':
      return 'Rate limit'
    case 'sent':
      return 'Успешно'
    default:
      return status
  }
}
