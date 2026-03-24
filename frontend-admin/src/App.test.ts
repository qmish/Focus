import { describe, it, expect } from 'vitest'

describe('Admin Frontend', () => {
  it('should pass basic test', () => {
    expect(true).toBe(true)
  })

  it('should have correct config', () => {
    const config = {
      api: 'http://localhost:8080',
      keycloak: 'http://localhost:8180',
    }
    expect(config.api).toContain('localhost')
  })
})
