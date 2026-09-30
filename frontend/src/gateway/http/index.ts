import { err, ok, type AppErrorCode, type OperationParams, type Result, type Transport } from '../types'

export interface HttpTransportOptions {
  readonly baseUrl: string
  readonly getAccessToken?: () => string | undefined
}

const ERROR_CODES: Record<string, true> = {
  INSUFFICIENT_PERMISSION: true,
  PERIOD_CLOSED: true,
  INVOICE_ALREADY_POSTED: true,
  PAYMENT_ALREADY_REVERSED: true,
  INVALID_ALLOCATION: true,
  UNBALANCED_JOURNAL: true,
  STOCK_INSUFFICIENT: true,
  PAYROLL_LOCKED: true,
  VALIDATION_FAILED: true,
  NOT_FOUND: true,
  CONFLICT: true,
  NETWORK: true,
  INTERNAL: true,
}

/**
 * HttpTransport talks to the EduPilot Site Server or the EduPilot Cloud API. It
 * is the Mode B and Mode C implementation of Transport, and it is deliberately
 * identical in shape to the desktop transport so the gateways above it do not
 * change.
 */
export function createHttpTransport(options: HttpTransportOptions): Transport {
  const base = options.baseUrl.replace(/\/$/, '')

  return {
    async call<T>(operation: string, params?: OperationParams): Promise<Result<T>> {
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      const token = options.getAccessToken?.()
      if (token !== undefined) {
        headers.Authorization = `Bearer ${token}`
      }

      let response: Response
      try {
        response = await fetch(`${base}/api/v1/${operation}`, {
          method: 'POST',
          headers,
          body: JSON.stringify(params ?? {}),
        })
      } catch (cause) {
        return err('NETWORK', cause instanceof Error ? cause.message : 'network failure')
      }

      const payload: unknown = await response.json().catch(() => undefined)

      if (!response.ok) {
        return toError(payload, response.status)
      }

      return ok(payload as T)
    },
  }
}

function toError(payload: unknown, status: number): Result<never> {
  if (typeof payload === 'object' && payload !== null && 'code' in payload && 'message' in payload) {
    const raw = payload as { code: string; message: string }
    const code: AppErrorCode = Object.prototype.hasOwnProperty.call(ERROR_CODES, raw.code)
      ? (raw.code as AppErrorCode)
      : 'INTERNAL'
    return err(code, raw.message)
  }

  return err(status === 403 ? 'INSUFFICIENT_PERMISSION' : 'INTERNAL', `request failed with status ${status}`)
}
