export const listen = async (_event?: string, _cb?: unknown) => () => {}
export const emit = async (_event?: string, _payload?: unknown) => {}
export const invoke = async <T = unknown>(_cmd?: string, _args?: unknown): Promise<T> => undefined as unknown as T
export default { listen, emit, invoke }
