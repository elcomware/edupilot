import { t } from '@/i18n'

export interface PlaceholderPageProps {
  readonly titleKey: string
}

/**
 * PlaceholderPage stands in for a module that has not been built yet. It is
 * deliberately honest about that, so nobody mistakes a navigation entry for a
 * working feature.
 */
export function PlaceholderPage({ titleKey }: PlaceholderPageProps) {
  return (
    <section>
      <h1 className="mb-3 text-[1.625rem] font-semibold">{t(titleKey)}</h1>
      <p className="m-0 text-ink-muted">{t('app.pending')}</p>
    </section>
  )
}
