import { createRoot } from 'react-dom/client'
import { RouterProvider } from 'react-router-dom'
import { router } from '@/app/router'
import { GatewayProvider } from '@/app/providers/GatewayProvider'
import { ThemeProvider } from '@/app/providers/ThemeProvider'
import { createGateway } from '@/gateway'
import { setLocale, type Locale } from '@/i18n'
import '@/design-system/theme.css'

export interface BootstrapOptions {
  readonly locale?: Locale
  readonly element?: HTMLElement | null
}

function detectLocale(): Locale {
  const stored = globalThis.localStorage?.getItem('edupilot.locale')
  if (stored === 'en' || stored === 'fr') {
    return stored
  }
  return navigator.language.toLowerCase().startsWith('fr') ? 'fr' : 'en'
}

export function bootstrap(options: BootstrapOptions = {}): void {
  const element = options.element ?? document.getElementById('root')
  if (element === null) {
    throw new Error('EduPilot could not start: #root is missing')
  }

  setLocale(options.locale ?? detectLocale())

  createRoot(element).render(
    <ThemeProvider>
      <GatewayProvider gateway={createGateway()}>
        <RouterProvider router={router} />
      </GatewayProvider>
    </ThemeProvider>,
  )
}
