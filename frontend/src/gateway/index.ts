import { createDesktopTransport, isDesktopRuntime } from './desktop'
import { createHttpTransport } from './http'
import { createPreviewTransport, isPreviewEnabled } from './preview'
import { createGatewayServices, type FinanceGateway } from './services'

export * from './types'
export { createGatewayServices, type FinanceGateway } from './services'
export { isPreviewEnabled } from './preview'

/**
 * The single place where the delivery adapter is chosen. The desktop edition
 * uses the Wails transport; a browser build uses the HTTP transport against the
 * Site Server or the Cloud API. Nothing above this file knows the difference.
 *
 * Preview comes first and only in a development build that asked for it by name,
 * so the interface can be worked on in a browser with no server running.
 */
export function createGateway(): FinanceGateway {
  if (isPreviewEnabled()) {
    return createGatewayServices(createPreviewTransport())
  }

  if (isDesktopRuntime()) {
    return createGatewayServices(createDesktopTransport())
  }

  return createGatewayServices(
    createHttpTransport({
      baseUrl: import.meta.env.VITE_API_BASE_URL ?? '',
    }),
  )
}
