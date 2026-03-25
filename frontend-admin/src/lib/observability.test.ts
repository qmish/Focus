import { describe, expect, it } from 'vitest'
import {
  authAuditStatusLabel,
  botStatusLabel,
  calendarOperationLabel,
  isDeliveryFailed,
} from './observability'

describe('observability utils', () => {
  it('detects failed delivery', () => {
    expect(isDeliveryFailed(false)).toBe(true)
    expect(isDeliveryFailed(true)).toBe(false)
  })

  it('maps bot status labels', () => {
    expect(botStatusLabel('failed')).toBe('Ошибка')
    expect(botStatusLabel('permission_denied')).toBe('Нет доступа')
    expect(botStatusLabel('rate_limited')).toBe('Rate limit')
    expect(botStatusLabel('sent')).toBe('Успешно')
    expect(botStatusLabel('custom')).toBe('custom')
  })

  it('maps auth audit status labels', () => {
    expect(authAuditStatusLabel('success')).toBe('Успешно')
    expect(authAuditStatusLabel('failed')).toBe('Ошибка')
  })

  it('maps calendar operation labels', () => {
    expect(calendarOperationLabel('create')).toBe('Создание')
    expect(calendarOperationLabel('update')).toBe('Обновление')
    expect(calendarOperationLabel('delete')).toBe('Удаление')
    expect(calendarOperationLabel('custom')).toBe('custom')
  })
})
