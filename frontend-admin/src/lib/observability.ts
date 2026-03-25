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

export function authAuditStatusLabel(status: string): string {
  return status === 'success' ? 'Успешно' : 'Ошибка'
}

export function calendarOperationLabel(operation: string): string {
  switch (operation) {
    case 'create':
      return 'Создание'
    case 'update':
      return 'Обновление'
    case 'delete':
      return 'Удаление'
    default:
      return operation
  }
}
