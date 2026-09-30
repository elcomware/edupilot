import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useGateway } from '@/app/providers/GatewayProvider'
import { t } from '@/i18n'
import type { AppError, Person, PersonRole, Relationship } from '@/gateway/types'
import { formatDate, initials, relationshipLabel, roleLabel, statusLabel } from './labels'

/**
 * PersonDetailPage answers the three questions a school asks about somebody:
 * who are they, what do they do here this year, and who is responsible for them.
 *
 * Roles and relationships are two separate sections on purpose. A role is what
 * the person is; a relationship is who they belong to. Keeping them apart is
 * what stops the old "person type" from creeping back in, where a teacher who
 * was also a parent had to be filed as one or the other.
 */
export function PersonDetailPage() {
  const { id } = useParams<{ id: string }>()
  const gateway = useGateway()

  const [person, setPerson] = useState<Person | null>(null)
  const [outgoing, setOutgoing] = useState<readonly Relationship[]>([])
  const [incoming, setIncoming] = useState<readonly Relationship[]>([])
  const [failure, setFailure] = useState<AppError | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (id === undefined) {
      return
    }
    let cancelled = false
    setLoading(true)

    void (async () => {
      const found = await gateway.people.get(id)
      if (cancelled) {
        return
      }
      if (!found.ok) {
        setFailure(found.error)
        setLoading(false)
        return
      }
      setPerson(found.value)
      setFailure(null)

      // Both directions are read, so one page can show "my children" and
      // "my guardians" without a second call or a client-side guess.
      const [out, into] = await Promise.all([
        gateway.people.listRelationships({ personId: id }),
        gateway.people.listRelationships({ personId: id, incoming: true }),
      ])
      if (cancelled) {
        return
      }
      setOutgoing(out.ok ? out.value : [])
      setIncoming(into.ok ? into.value : [])
      setLoading(false)
    })()

    return () => {
      cancelled = true
    }
  }, [gateway, id])

  if (loading) {
    return (
      <section>
        <p className="m-0 text-ink-muted">{t('app.loading')}</p>
      </section>
    )
  }

  if (failure !== null) {
    const key = `errors.${failure.code}`
    const translated = t(key)
    return (
      <section>
        <h1 className="mb-3 text-[1.625rem] font-semibold">{t('people.person')}</h1>
        <div
          role="alert"
          className="flex items-center justify-between gap-3 rounded-[var(--radius-panel)] bg-negative-subtle px-3 py-2.5 text-negative"
        >
          <span>{translated === key ? failure.message : translated}</span>
          <Link to="/people" className="text-sm text-negative underline">
            {t('people.backToList')}
          </Link>
        </div>
      </section>
    )
  }

  if (person === null) {
    return null
  }

  return (
    <section>
      <Link to="/people" className="text-xs text-ink-muted no-underline hover:text-accent">
        ← {t('people.backToList')}
      </Link>

      <header className="mt-2 mb-5 flex items-center gap-3">
        <span
          aria-hidden="true"
          className="inline-flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-accent-subtle text-base font-semibold text-accent"
        >
          {initials(person.displayName)}
        </span>
        <div>
          <h1 className="m-0 text-[1.625rem] font-semibold">{person.displayName}</h1>
          <p className="mt-0.5 mb-0 text-sm text-ink-muted">
            {person.isActive ? t('people.active') : t('people.inactive')}
            {person.preferredName !== '' && ` · ${t('people.knownAs', { name: person.preferredName })}`}
          </p>
        </div>
      </header>

      <div className="grid grid-cols-[repeat(auto-fit,minmax(320px,1fr))] gap-4">
        <Panel title={t('people.details')}>
          <Details
            rows={[
              [t('people.firstName'), person.firstName],
              [t('people.middleName'), person.middleName],
              [t('people.lastName'), person.lastName],
              [t('people.dateOfBirth'), formatDate(person.dateOfBirth)],
              [t('people.gender'), person.gender],
              [t('people.nationality'), person.nationality],
              [t('people.nationalId'), person.nationalId],
            ]}
          />
        </Panel>

        <Panel title={t('people.contact')}>
          <Details
            rows={[
              [t('people.email'), person.email],
              [t('people.phone'), person.phone],
              [t('people.secondaryPhone'), person.secondaryPhone],
              [t('people.address'), joinAddress(person.address)],
            ]}
          />
        </Panel>
      </div>

      <Panel title={t('people.roles')} className="mt-4">
        {person.roles.length === 0 ? (
          <p className="m-0 text-ink-muted">{t('people.noRoles')}</p>
        ) : (
          <ul className="m-0 flex list-none flex-col gap-3 p-0">
            {person.roles.map((role) => (
              <RoleCard key={role.id} role={role} />
            ))}
          </ul>
        )}
      </Panel>

      <div className="mt-4 grid grid-cols-[repeat(auto-fit,minmax(320px,1fr))] gap-4">
        <Panel title={t('people.relationshipsOut')}>
          <RelationshipList relationships={outgoing} direction="out" />
        </Panel>
        <Panel title={t('people.relationshipsIn')}>
          <RelationshipList relationships={incoming} direction="in" />
        </Panel>
      </div>
    </section>
  )
}

function RoleCard({ role }: { readonly role: PersonRole }) {
  return (
    <li className="rounded-[var(--radius-panel)] border border-line bg-surface-muted px-3 py-2">
      <div className="mb-1 flex flex-wrap items-baseline gap-2">
        <span className="text-sm font-semibold">{roleLabel(role.role)}</span>
        {role.isCurrent ? (
          <span className="rounded-[var(--radius-control)] bg-positive-subtle px-1.5 py-0.5 text-[0.6875rem] font-medium text-positive">
            {t('people.thisYear')}
          </span>
        ) : null}
        {(role.startsOn !== '' || role.endsOn !== '') && (
          <span className="text-xs text-ink-muted">
            {formatDate(role.startsOn)} → {role.endsOn === '' ? t('common.present') : formatDate(role.endsOn)}
          </span>
        )}
      </div>

      {role.student !== undefined && (
        <Details
          compact
          rows={[
            [t('people.studentNumber'), role.student.studentNumber],
            [t('people.status'), statusLabel(role.student.status)],
            [t('people.admissionDate'), formatDate(role.student.admissionDate)],
            [t('people.previousSchool'), role.student.previousSchool],
            [t('people.boarding'), role.student.isBoarding ? t('common.yes') : t('common.no')],
          ]}
        />
      )}

      {role.employee !== undefined && (
        <Details
          compact
          rows={[
            [t('people.employeeNumber'), role.employee.employeeNumber],
            [t('people.jobTitle'), role.employee.jobTitle],
            [t('people.department'), role.employee.department],
            [t('people.hiredOn'), formatDate(role.employee.hiredOn)],
            // A leaver keeps the record, so the end date is shown rather than the
            // employment being hidden.
            [t('people.endedOn'), role.employee.endedOn === '' ? t('common.present') : formatDate(role.employee.endedOn)],
            [t('people.contractType'), t(`people.contractTypes.${role.employee.contractType}`)],
            [t('people.payrollGroup'), role.employee.payrollGroup],
          ]}
        />
      )}

      {role.guardian !== undefined && (
        <ul className="m-0 flex list-none flex-wrap gap-x-4 gap-y-1 p-0 text-sm">
          <Right label={t('people.rights.emergencyContact')} on={role.guardian.isEmergencyContact} />
          <Right label={t('people.rights.mayCollect')} on={role.guardian.mayCollectStudent} />
          <Right label={t('people.rights.billingContact')} on={role.guardian.isBillingContact} />
          <Right label={t('people.rights.medical')} on={role.guardian.canAuthoriseMedical} />
        </ul>
      )}
    </li>
  )
}

function Right({ label, on }: { readonly label: string; readonly on: boolean }) {
  return (
    <li className={on ? 'text-ink' : 'text-ink-muted'}>
      <span aria-hidden="true" className="mr-1">
        {on ? '✓' : '·'}
      </span>
      {label}
    </li>
  )
}

/**
 * An edge is shown with the other end linked, but only by identifier: resolving
 * every name would mean one call per edge, and a parent with four children at
 * this school would mean four calls to draw one line.
 */
function RelationshipList({
  relationships,
  direction,
}: {
  readonly relationships: readonly Relationship[]
  readonly direction: 'in' | 'out'
}) {
  if (relationships.length === 0) {
    return <p className="m-0 text-ink-muted">{t('people.noRelationships')}</p>
  }
  return (
    <ul className="m-0 flex list-none flex-col gap-1.5 p-0">
      {relationships.map((relationship) => {
        const otherId = direction === 'out' ? relationship.toPersonId : relationship.fromPersonId
        return (
          <li key={relationship.id} className="flex items-baseline gap-2 text-sm">
            <span className={relationship.isActive ? 'text-ink' : 'text-ink-muted line-through'}>
              {relationshipLabel(relationship.type, relationship.role)}
            </span>
            <Link to={`/people/${otherId}`} className="text-accent">
              {t('people.viewPerson')}
            </Link>
            {relationship.isPrimary && (
              <span className="rounded-[var(--radius-control)] bg-accent-subtle px-1.5 py-0.5 text-[0.6875rem] font-medium text-accent">
                {t('people.primary')}
              </span>
            )}
          </li>
        )
      })}
    </ul>
  )
}

function Panel({
  title,
  children,
  className = '',
}: {
  readonly title: string
  readonly children: React.ReactNode
  readonly className?: string
}) {
  return (
    <article className={`rounded-[var(--radius-panel)] border border-line bg-surface p-4 ${className}`}>
      <h2 className="mb-3 text-base font-semibold">{title}</h2>
      {children}
    </article>
  )
}

function Details({
  rows,
  compact = false,
}: {
  readonly rows: readonly (readonly [string, string])[]
  readonly compact?: boolean
}) {
  const shown = rows.filter(([, value]) => value !== '')
  if (shown.length === 0) {
    return <p className="m-0 text-ink-muted">{t('people.notRecorded')}</p>
  }
  return (
    <dl className={`m-0 grid grid-cols-[max-content_1fr] gap-x-4 ${compact ? 'gap-y-0.5 text-sm' : 'gap-y-1'}`}>
      {shown.map(([label, value]) => (
        <div key={label} className="contents">
          <dt className="text-ink-muted">{label}</dt>
          <dd className="m-0">{value}</dd>
        </div>
      ))}
    </dl>
  )
}

function joinAddress(address: Person['address']): string {
  return [address.line1, address.line2, address.postalCode, address.city, address.country]
    .filter((part) => part !== '')
    .join(', ')
}
