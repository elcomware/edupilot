import { t } from '@/i18n'
import type { PersonRoleType, RelationshipRole, RelationshipType } from '@/gateway/types'

/**
 * A calendar date arrives as `YYYY-MM-DD` and is shown in the reader's own
 * format. The parts are read in UTC rather than handed to `new Date(string)`,
 * because that constructor reads a bare date as local time and would show
 * somebody born on the 1st as born on the 31st of the previous month west of
 * Greenwich.
 */
export function formatDate(iso: string): string {
  if (iso === '') {
    return '—'
  }
  const parts = /^(\d{4})-(\d{2})-(\d{2})/.exec(iso)
  if (parts === null) {
    return iso
  }
  const [, year, month, day] = parts
  return new Date(Date.UTC(Number(year), Number(month) - 1, Number(day))).toLocaleDateString(undefined, {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    timeZone: 'UTC',
  })
}

export function roleLabel(role: PersonRoleType): string {
  return t(`people.roles.${role}`)
}

/**
 * A relationship is read as a sentence, "MOTHER of" rather than
 * "GUARDIAN_OF / MOTHER", because that is the question a bursar is asking.
 */
export function relationshipLabel(type: RelationshipType, role: RelationshipRole): string {
  if (role === '') {
    return t(`people.relationshipTypes.${type}`)
  }
  return t(`people.relationshipRoles.${role}`)
}

export function statusLabel(status: string): string {
  const key = `people.studentStatuses.${status}`
  const translated = t(key)
  return translated === key ? status : translated
}

/**
 * Two letters for the avatar tile, taken from the first and last name. A person
 * with one name only still gets a tile rather than a blank one.
 */
export function initials(displayName: string): string {
  const parts = displayName.trim().split(/\s+/).filter((part) => part !== '')
  if (parts.length === 0) {
    return '?'
  }
  const first = parts[0]?.[0] ?? ''
  const last = parts.length > 1 ? (parts[parts.length - 1]?.[0] ?? '') : ''
  return (first + last).toUpperCase()
}

/** A single word for "this person holds two roles", used in list rows. */
export function roleSummary(roles: readonly { role: PersonRoleType }[]): string {
  if (roles.length === 0) {
    return t('people.noRoles')
  }
  return roles.map((role) => roleLabel(role.role)).join(' · ')
}
