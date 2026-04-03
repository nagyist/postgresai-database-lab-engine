import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

vi.mock('config/env', () => ({ WS_URL_PREFIX: '' }))

import { initWS } from './initWS'

describe('initWS', () => {
  let MockWebSocket: ReturnType<typeof vi.fn>

  beforeEach(() => {
    MockWebSocket = vi.fn()
    vi.stubGlobal('WebSocket', MockWebSocket)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('converts http location to ws URL', () => {
    vi.stubGlobal('window', { location: { href: 'http://localhost:3001/' } })
    initWS('/path/to/ws', 'token123')
    expect(MockWebSocket).toHaveBeenCalledWith(
      'ws://localhost:3001/path/to/ws?token=token123',
    )
  })

  it('converts https location to wss URL', () => {
    vi.stubGlobal('window', { location: { href: 'https://example.com/' } })
    initWS('/path/to/ws', 'token456')
    expect(MockWebSocket).toHaveBeenCalledWith(
      'wss://example.com/path/to/ws?token=token456',
    )
  })
})

describe('initWS with WS_URL_PREFIX', () => {
  let MockWebSocket: ReturnType<typeof vi.fn>

  beforeEach(() => {
    MockWebSocket = vi.fn()
    vi.stubGlobal('WebSocket', MockWebSocket)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('prepends WS_URL_PREFIX to path', async () => {
    vi.resetModules()
    vi.doMock('config/env', () => ({ WS_URL_PREFIX: '/api/v1' }))
    const { initWS: initWSPrefixed } = await import('./initWS')
    vi.stubGlobal('window', { location: { href: 'https://example.com/' } })
    initWSPrefixed('/ws', 'tok')
    expect(MockWebSocket).toHaveBeenCalledWith(
      'wss://example.com/api/v1/ws?token=tok',
    )
  })
})
