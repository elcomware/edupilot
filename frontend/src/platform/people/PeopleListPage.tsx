import { useEffect, useMemo, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { useGateway } from '@/app/providers/GatewayProvider'
import { t } from '@/i18n'
import type { AppError, ListPeopleRequest, Page, Person, PersonRoleType } from '@/gateway/types'
import { PERSON_ROLES } from '@/gateway/types'
import { initials, roleLabel } from './labels'

const PAGE_SIZE = 25

/**
 * PeopleListPage is the school's directory: every person the school has a record
 * for, with the roles they hold this year beside their name.
 *
 * The search is sent to the backend, because a school with a thousand pupils
 * should not have to download all of them to find one. The role filter is
 * applied to the page that was loaded, and the count above the table says so,
 * because filtering a page client-side and calling it a school-wide result would
 * be a lie the moment a school got big.
 */
export function PeopleListPage() {
  const gateway = useGateway()

  const [search, setSearch] = useState('')
  const [committedSearch, setCommittedSearch] = useState('')
  const [roleFilter, setRoleFilter] = useState<PersonRoleType | ''>('')
  const [people, setPeople] = useState<Page<Person> | null>(null)
  const [failure, setFailure] = useState<AppError | null>(null)
  const [loading, setLoading] = useState(true)

  // The search box is debounced so a burst of keystrokes is one round trip.
  const debounce = useRef<number | undefined>(undefined)
  useEffect(() => {
    window.clearTimeout(debounce.current)
    debounce.current = window.setTimeout(() => setCommittedSearch(search.trim()), 250)
    return () => window.clearTimeout(debounce.current)
  }, [search])

  useEffect(() => {
    let cancelled = false
    setLoading(true)

    void (async () => {
      // The search term is left out of the request entirely when it is empty,
      // rather than sent as an empty string that the backend would have to
      // decide whether it means "everyone" or "nobody".
      const request: ListPeopleRequest = { limit: PAGE_SIZE, offset: 0 }
      const result = committedSearch === '' ? await gateway.people.list(request) : await gateway.people.list({ ...request, search: committedSearch })
      if (cancelled) {
        return
      }
      if (!result.ok) {
        setFailure(result.error)
      } else {
        setPeople(result.value)
        setFailure(null)
      }
      setLoading(false)
    })()

    return () => {
      cancelled = true
    }
  }, [gateway, committedSearch])

  const visible = useMemo(() => {
    if (people === null) {
      return []
    }
    if (roleFilter === '') {
      return people.items
    }
    return people.items.filter((person) => person.roles.some((role) => role.role === roleFilter))
  }, [people, roleFilter])

  return (
    <section>
      <header className="mb-4">
        <h1 className="m-0 text-[1.625rem] font-semibold">{t('people.title')}</h1>
        <p className="mt-1 mb-0 text-ink-muted">{t('people.subtitle')}</p>
      </header>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <input
          type="search"
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          placeholder={t('people.searchPlaceholder')}
          aria-label={t('people.searchPlaceholder')}
          className="h-8 w-64 rounded-[var(--radius-control)] border border-line bg-surface px-2.5 text-sm text-ink placeholder:text-ink-muted focus:border-accent focus:outline-none"
        />

        <div className="flex flex-wrap items-center gap-1" role="group" aria-label={t('people.filterByRole')}>
          <FilterChip
            label={t('people.allRoles')}
            active={roleFilter === ''}
            onClick={() => setRoleFilter('')}
          />
          {PERSON_ROLES.map((role) => (
            <FilterChip
              key={role}
              label={roleLabel(role)}
              active={roleFilter === role}
              onClick={() => setRoleFilter(roleFilter === role ? '' : role)}
            />
          ))}
        </div>
      </div>

      {loading && <p className="mt-4 text-ink-muted">{t('app.loading')}</p>}

      {failure !== null && <FailureBanner failure={failure} />}

      {!loading && failure === null && (
        <>
          <p className="mt-0 mb-2 text-xs text-ink-muted">
            {roleFilter === ''
              ? t('people.showingCount', { count: people?.total ?? 0 })
              : t('people.showingOfCount', { shown: visible.length, count: people?.total ?? 0 })}
          </p>

          {visible.length === 0 ? (
            <p className="m-0 text-ink-muted">
              {people !== null && people.total === 0 ? t('people.noPeople') : t('people.noMatches')}
            </p>
          ) : (
            <div className="overflow-x-auto rounded-[var(--radius-panel)] border border-line bg-surface">
              <table className="w-full border-collapse">
                <thead>
                  <tr className="border-b border-line text-left text-xs text-ink-muted">
                    <th scope="col" className="px-3 py-2 font-medium">
                      {t('people.name')}
                    </th>
                    <th scope="col" className="px-3 py-2 font-medium">
                      {t('people.roles')}
                    </th>
                    <th scope="col" className="px-3 py-2 font-medium">
                      {t('people.contact')}
                    </th>
                    <th scope="col" className="px-3 py-2 font-medium">
                      {t('people.city')}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {visible.map((person) => (
                    <tr key={person.id} className="border-b border-line last:border-none hover:bg-surface-muted">
                      <td className="px-3 py-1.5">
                        <Link
                          to={`/people/${person.id}`}
                          className="flex items-center gap-2.5 text-ink no-underline hover:text-accent"
                        >
                          <span
                            aria-hidden="true"
                            className="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-accent-subtle text-[0.6875rem] font-semibold text-accent"
                          >
                            {initials(person.displayName)}
                          </span>
                          <span className={person.isActive ? '' : 'text-ink-muted line-through'}>
                            {person.displayName}
                          </span>
                        </Link>
                      </td>
                      <td className="px-3 py-1.5">
                        {person.roles.length === 0 ? (
                          <span className="text-ink-muted">{t('people.noRoles')}</span>
                        ) : (
                          <span className="flex flex-wrap gap-1">
                            {person.roles.map((role) => (
                              <span
                                key={role.id}
                                className="rounded-[var(--radius-control)] bg-accent-subtle px-1.5 py-0.5 text-[0.6875rem] font-medium text-accent"
                              >
                                {roleLabel(role.role)}
                              </span>
                            ))}
                          </span>
                        )}
                      </td>
                      <td className="px-3 py-1.5 text-ink-muted">
                        {person.email || person.phone || '—'}
                      </td>
                      <td className="px-3 py-1.5 text-ink-muted">{person.address.city || '—'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}
    </section>
  )
}

interface FilterChipProps {
  readonly label: string
  readonly active: boolean
  readonly onClick: () => void
}

function FilterChip({ label, active, onClick }: FilterChipProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={`h-7 rounded-[var(--radius-control)] border px-2.5 text-xs transition-colors ${
        active
          ? 'border-accent bg-accent-subtle font-medium text-accent'
          : 'border-line bg-surface text-ink-muted hover:border-accent hover:text-accent'
      }`}
    >
      {label}
    </button>
  )
}

/**
 * A failure is shown by its code so the wording is identical in every language;
 * the backend message is only a fallback for codes not yet translated.
 */
function FailureBanner({ failure }: { readonly failure: AppError }) {
  const key = `errors.${failure.code}`
  const translated = t(key)
  return (
    <div
      role="alert"
      className="mt-4 flex items-center justify-between gap-3 rounded-[var(--radius-panel)] bg-negative-subtle px-3 py-2.5 text-negative"
    >
      <span>{translated === key ? failure.message : translated}</span>
    </div>
  )
}
