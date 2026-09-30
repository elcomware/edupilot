import { useCallback, useEffect, useState } from 'react'
import { useGateway } from '@/app/providers/GatewayProvider'
import { t } from '@/i18n'
import type { AppError, Campus, Organisation, Page } from '@/gateway/types'

/**
 * DashboardPage is the first screen backed by real data. It shows the tenant
 * and its campuses, which is enough to prove the whole chain — React, gateway,
 * transport, application service, SQLite — works in this delivery mode.
 */
export function DashboardPage() {
  const gateway = useGateway()

  const [organisation, setOrganisation] = useState<Organisation | null>(null)
  const [campuses, setCampuses] = useState<Page<Campus> | null>(null)
  const [failure, setFailure] = useState<AppError | null>(null)
  const [loading, setLoading] = useState(true)

  const load = useCallback(() => {
    let cancelled = false
    setLoading(true)

    void (async () => {
      const school = await gateway.organisation.getCurrent()
      if (cancelled) {
        return
      }
      if (!school.ok) {
        setFailure(school.error)
        setLoading(false)
        return
      }
      setOrganisation(school.value)
      setFailure(null)

      const page = await gateway.organisation.listCampuses({ limit: 20, offset: 0 })
      if (cancelled) {
        return
      }
      if (!page.ok) {
        setFailure(page.error)
      } else {
        setCampuses(page.value)
      }
      setLoading(false)
    })()

    return () => {
      cancelled = true
    }
  }, [gateway])

  useEffect(load, [load])

  return (
    <section>
      <h1 className="mb-3 text-[1.625rem] font-semibold">{t('dashboard.title')}</h1>
      <p className="m-0 text-ink-muted">{t('dashboard.subtitle')}</p>

      {loading && <p className="mt-4 text-ink-muted">{t('app.loading')}</p>}

      {failure !== null && (
        <div
          role="alert"
          className="mt-4 flex items-center justify-between gap-3 rounded-[var(--radius-panel)] bg-negative-subtle px-3 py-2.5 text-negative"
        >
          <span>{describeError(failure)}</span>
          <button
            type="button"
            onClick={load}
            className="h-7 rounded-[var(--radius-control)] border border-line bg-surface px-3 text-xs text-ink hover:border-accent hover:text-accent"
          >
            {t('common.retry')}
          </button>
        </div>
      )}

      {organisation !== null && (
        <div className="mt-6 grid grid-cols-[repeat(auto-fit,minmax(320px,1fr))] gap-4">
          <article className="rounded-[var(--radius-panel)] border border-line bg-surface p-4">
            <h2 className="mb-3 text-base font-semibold">{organisation.name}</h2>
            <dl className="m-0 grid grid-cols-[max-content_1fr] gap-x-4 gap-y-1">
              <dt className="text-ink-muted">{t('dashboard.legalName')}</dt>
              <dd className="m-0">{organisation.legalName}</dd>
              <dt className="text-ink-muted">{t('dashboard.currency')}</dt>
              <dd className="m-0 tabular">{organisation.currency}</dd>
              <dt className="text-ink-muted">{t('dashboard.country')}</dt>
              <dd className="m-0">{organisation.country}</dd>
              <dt className="text-ink-muted">{t('dashboard.locale')}</dt>
              <dd className="m-0">{organisation.defaultLocale}</dd>
            </dl>
          </article>

          <article className="rounded-[var(--radius-panel)] border border-line bg-surface p-4">
            <h2 className="mb-3 text-base font-semibold">
              {t('dashboard.campuses')}
              {campuses !== null && <span className="font-normal text-ink-muted"> {campuses.total}</span>}
            </h2>
            {campuses !== null && campuses.items.length === 0 && (
              <p className="m-0 text-ink-muted">{t('dashboard.noCampuses')}</p>
            )}
            {campuses !== null && campuses.items.length > 0 && (
              <table className="w-full border-collapse">
                <thead>
                  <tr className="border-b border-line text-left text-xs text-ink-muted">
                    <th scope="col" className="px-2 py-1 font-medium">
                      {t('dashboard.campusName')}
                    </th>
                    <th scope="col" className="px-2 py-1 font-medium">
                      {t('dashboard.campusCode')}
                    </th>
                    <th scope="col" className="px-2 py-1 font-medium">
                      {t('dashboard.campusAddress')}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {campuses.items.map((campus) => (
                    <tr key={campus.id} className="border-b border-line last:border-none">
                      <td className="px-2 py-1">{campus.name}</td>
                      <td className="tabular px-2 py-1">{campus.code}</td>
                      <td className="px-2 py-1">{campus.address}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </article>
        </div>
      )}
    </section>
  )
}

/**
 * A failure is shown by its code, so the wording is the same in every language.
 * The backend message is only a fallback for codes the interface has not
 * translated yet.
 */
function describeError(failure: AppError): string {
  const translated = t(`errors.${failure.code}`)
  return translated === `errors.${failure.code}` ? failure.message : translated
}
