import { NavLink, Outlet } from 'react-router-dom'
import { t } from '@/i18n'
import { isPreviewEnabled } from '@/gateway'
import { NAVIGATION } from './navigation'
import { SkinSwitcher } from './SkinSwitcher'

/**
 * AppShell is the one layout every page renders inside. It is written entirely
 * with Tailwind utilities and semantic colour names, so it follows whichever
 * skin and mode are active without a single conditional class.
 */
export function AppShell() {
  return (
    <div className="grid h-full grid-cols-[var(--spacing-sidebar)_1fr] grid-rows-[var(--spacing-header)_1fr] print:grid-cols-1 print:grid-rows-1">
      <header className="col-span-2 row-start-1 flex items-center justify-between gap-3 bg-accent px-4 text-on-accent print:hidden">
        <div className="flex items-baseline gap-3">
          <span className="text-base font-semibold">{t('app.name')}</span>
          <span className="text-xs opacity-85">{t('app.tagline')}</span>
        </div>
        <SkinSwitcher />
      </header>

      {isPreviewEnabled() && (
        <div
          role="status"
          className="col-span-2 row-start-1 -mt-[var(--spacing-header)] flex h-[var(--spacing-header)] items-center justify-center gap-2 bg-warning-subtle px-4 text-xs font-medium text-warning"
        >
          {t('app.previewBanner')}
        </div>
      )}

      <nav className="row-start-2 overflow-y-auto border-r border-line bg-surface py-3 print:hidden">
        {NAVIGATION.map((section) => (
          <div key={section.path} className="mb-3">
            <NavLink
              to={section.path}
              className={({ isActive }) =>
                `block px-4 py-0.5 text-xs font-semibold tracking-wider uppercase no-underline transition-colors ${
                  isActive ? 'bg-accent-subtle text-accent' : 'text-ink-muted hover:text-accent'
                }`
              }
            >
              {t(section.key)}
            </NavLink>
            {section.items.length > 0 && (
              <ul className="m-0 list-none p-0">
                {section.items.map((item) => (
                  <li key={item.path}>
                    <NavLink
                      to={item.path}
                      className={({ isActive }) =>
                        `block px-4 py-0.5 pl-5 text-[0.8125rem] no-underline transition-colors ${
                          isActive
                            ? 'bg-accent-subtle font-medium text-accent'
                            : 'text-ink hover:bg-accent-subtle hover:text-accent'
                        }`
                      }
                    >
                      {t(item.key)}
                    </NavLink>
                  </li>
                ))}
              </ul>
            )}
          </div>
        ))}
      </nav>

      <main className="row-start-2 overflow-y-auto p-6">
        <Outlet />
      </main>
    </div>
  )
}
