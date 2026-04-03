import { describe, it, expect, vi, beforeEach } from 'vitest'

describe('env', () => {
  beforeEach(() => {
    vi.resetModules()
    vi.unstubAllEnvs()
  })

  it('exports API_URL_PREFIX from VITE_API_URL_PREFIX', async () => {
    vi.stubEnv('VITE_API_URL_PREFIX', '/api/prefix')
    const { API_URL_PREFIX } = await import('./env')
    expect(API_URL_PREFIX).toBe('/api/prefix')
  })

  it('falls back to empty string when VITE_API_URL_PREFIX is missing', async () => {
    const { API_URL_PREFIX } = await import('./env')
    expect(API_URL_PREFIX).toBe('')
  })

  it('exports WS_URL_PREFIX from VITE_WS_URL_PREFIX', async () => {
    vi.stubEnv('VITE_WS_URL_PREFIX', '/ws/prefix')
    const { WS_URL_PREFIX } = await import('./env')
    expect(WS_URL_PREFIX).toBe('/ws/prefix')
  })

  it('falls back to empty string when VITE_WS_URL_PREFIX is missing', async () => {
    const { WS_URL_PREFIX } = await import('./env')
    expect(WS_URL_PREFIX).toBe('')
  })

  it('exports BUILD_TIMESTAMP from process.env', async () => {
    vi.stubEnv('BUILD_TIMESTAMP', '1700000000000')
    const { BUILD_TIMESTAMP } = await import('./env')
    expect(BUILD_TIMESTAMP).toBe('1700000000000')
  })

  it('exports NODE_ENV from import.meta.env.MODE', async () => {
    const { NODE_ENV } = await import('./env')
    expect(NODE_ENV).toBe('test')
  })
})
