import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { useSearchStore } from './searchStore'

// Мокаем searchClient на уровне модуля.
vi.mock('../lib/searchClient', () => {
  const SearchAbortError = class extends Error {
    constructor() {
      super('search aborted')
      this.name = 'SearchAbortError'
    }
  }
  return {
    SearchAbortError,
    searchGlobal: vi.fn(),
  }
})

// eslint-disable-next-line @typescript-eslint/no-require-imports
import { searchGlobal } from '../lib/searchClient'

const mocked = searchGlobal as unknown as ReturnType<typeof vi.fn>

const sample = {
  users: [],
  rooms: [],
  messages: [],
  files: [],
  meetings: [],
  took_ms: 0,
  query: 'q',
}

describe('searchStore', () => {
  beforeEach(() => {
    useSearchStore.setState({
      isOpen: false,
      query: '',
      results: null,
      loading: false,
      error: null,
      _abort: null,
      _lastQuery: '',
    })
    mocked.mockReset()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('rejects queries shorter than 2 chars', async () => {
    await useSearchStore.getState().search('a')
    expect(mocked).not.toHaveBeenCalled()
    expect(useSearchStore.getState().results).not.toBeNull()
    expect(useSearchStore.getState().loading).toBe(false)
  })

  it('calls API for valid query and stores result', async () => {
    mocked.mockResolvedValueOnce({ ...sample, query: 'foo' })
    await useSearchStore.getState().search('foo')
    expect(mocked).toHaveBeenCalledTimes(1)
    expect(useSearchStore.getState().results?.query).toBe('foo')
    expect(useSearchStore.getState().loading).toBe(false)
  })

  it('aborts stale request and ignores its result', async () => {
    let resolveFirst!: (v: typeof sample) => void
    let resolveSecond!: (v: typeof sample) => void

    mocked.mockImplementationOnce(
      (params: { signal?: AbortSignal }) =>
        new Promise<typeof sample>((resolve, reject) => {
          resolveFirst = resolve
          if (params.signal) {
            params.signal.addEventListener('abort', () => {
              const err: Error & { name?: string } = new Error('aborted')
              err.name = 'AbortError'
              reject(err)
            })
          }
        }),
    )
    mocked.mockImplementationOnce(
      () =>
        new Promise<typeof sample>((resolve) => {
          resolveSecond = resolve
        }),
    )

    const p1 = useSearchStore.getState().search('foo')
    const p2 = useSearchStore.getState().search('bar')

    resolveFirst({ ...sample, query: 'foo' })
    resolveSecond({ ...sample, query: 'bar' })

    await Promise.allSettled([p1, p2])

    expect(useSearchStore.getState().results?.query).toBe('bar')
    expect(useSearchStore.getState().loading).toBe(false)
  })

  it('sets error when API throws', async () => {
    mocked.mockRejectedValueOnce(new Error('boom'))
    await useSearchStore.getState().search('something')
    expect(useSearchStore.getState().error).toBe('boom')
    expect(useSearchStore.getState().loading).toBe(false)
  })

  it('open/close toggles isOpen', () => {
    useSearchStore.getState().open()
    expect(useSearchStore.getState().isOpen).toBe(true)
    useSearchStore.getState().close()
    expect(useSearchStore.getState().isOpen).toBe(false)
  })

  it('reset clears state', () => {
    useSearchStore.setState({ query: 'foo', results: { ...sample }, error: 'x' })
    useSearchStore.getState().reset()
    const s = useSearchStore.getState()
    expect(s.query).toBe('')
    expect(s.results).toBeNull()
    expect(s.error).toBeNull()
  })
})
