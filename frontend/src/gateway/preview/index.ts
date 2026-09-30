import { err, ok, type OperationParams, type Result, type Transport } from '../types'
import type { Campus, Household, Organisation, Person, PersonRole, Relationship, Term } from '../types'
import { FIXTURE_ACADEMIC_YEAR, FIXTURE_HOUSEHOLDS, FIXTURE_PEOPLE, FIXTURE_RELATIONSHIPS, FIXTURE_TERMS } from './fixtures'

/**
 * PreviewTransport serves the People screens from fixtures so the interface can
 * be built and reviewed in a browser, where there is no Wails runtime and no
 * server to talk to.
 *
 * It is not a mock in the sense of pretending to be a backend: it answers the
 * same operations with the same envelope the Go transport returns, so a screen
 * that works here works against SQLite for the same reason. What it does not do
 * is persist anything, and it must never be reachable in a shipped build — see
 * isPreviewEnabled below and the banner in the shell.
 */
export function createPreviewTransport(): Transport {
  return {
    async call<T>(operation: string, params?: OperationParams): Promise<Result<T>> {
      const request = (params ?? {}) as Record<string, unknown>
      const answered = answer(operation, request)
      if (answered === undefined) {
        return err('INTERNAL', `preview transport has no answer for ${operation}`)
      }
      // The answer is already a Result, including when it is a failure. Wrapping
      // it again would hand a success envelope containing a failure, which is
      // exactly the shape a screen cannot tell apart from real data.
      return answered as Result<T>
    },
  }
}

/**
 * Preview data is only ever served in a development build that explicitly asked
 * for it. Both conditions are required, so a forgotten flag in a production
 * environment still cannot reach this code.
 */
export function isPreviewEnabled(): boolean {
  return import.meta.env.DEV && import.meta.env.VITE_PREVIEW === '1'
}

function answer(operation: string, request: Record<string, unknown>): Result<unknown> | undefined {
  switch (operation) {
    case 'platform.organisation.App.GetCurrent':
      return ok(organisation())

    case 'platform.campus.App.List':
      return ok({ items: campuses(), total: campuses().length })

    case 'platform.academic.App.ListYears':
      return ok({ items: [FIXTURE_ACADEMIC_YEAR], total: 1 })

    case 'platform.academic.App.GetCurrentYear':
      return ok(FIXTURE_ACADEMIC_YEAR)

    case 'platform.academic.App.ListTerms':
      return ok(FIXTURE_TERMS)

    case 'platform.people.App.ListPeople':
      return ok(listPeople(request))

    case 'platform.people.App.GetPerson': {
      const person = personById(asString(request.id))
      // A record that does not exist is reported as not found, the same as the
      // real transport, so a screen's empty and error states are exercised here
      // too.
      return person === undefined ? notFound('person') : ok(person)
    }

    case 'platform.people.App.ListRelationships':
      return ok(relationshipsFor(asString(request.personId), request.incoming === true))

    case 'platform.people.App.ListHouseholds':
      return ok(FIXTURE_HOUSEHOLDS)

    default:
      return undefined
  }
}

function organisation(): Organisation {
  return {
    id: 'org-1',
    name: 'Groupe Scolaire Akwa',
    legalName: 'Groupe Scolaire Akwa SARL',
    currency: 'XAF',
    country: 'CM',
    defaultLocale: 'fr',
  }
}

function campuses(): readonly Campus[] {
  return [
    { id: 'campus-1', organisationId: 'org-1', name: 'Campus Akwa', code: 'AKW', address: 'Rue de la Joie, Akwa' },
    { id: 'campus-2', organisationId: 'org-1', name: 'Campus Bonapriso', code: 'BON', address: 'Avenue de la Liberté, Bonapriso' },
  ]
}

/**
 * The list is filtered and paged here so the screen is exercised the way it will
 * be in production: a search term, a page size, and a total that is larger than
 * the number of rows on the page.
 */
function listPeople(request: Record<string, unknown>): { items: Person[]; total: number } {
  const search = (asString(request.search) ?? '').toLowerCase()
  const matched = search === ''
    ? FIXTURE_PEOPLE
    : FIXTURE_PEOPLE.filter((person) =>
        [person.displayName, person.email, person.phone].some((field) => field.toLowerCase().includes(search)),
      )

  const limit = asNumber(request.limit) ?? matched.length
  const offset = asNumber(request.offset) ?? 0
  return { items: matched.slice(offset, offset + limit), total: matched.length }
}

/**
 * A person read individually carries their roles, which is the whole point of
 * the person screen: a role is useless without the profile that belongs to it.
 */
function personById(id: string | undefined): Person | undefined {
  if (id === undefined) {
    return undefined
  }
  const person = FIXTURE_PEOPLE.find((candidate) => candidate.id === id)
  if (person === undefined) {
    return undefined
  }
  return { ...person, roles: rolesFor(id) }
}

function rolesFor(personId: string): readonly PersonRole[] {
  return FIXTURE_PEOPLE.flatMap((person) => person.roles).filter((role) => role.personId === personId)
}

function relationshipsFor(personId: string | undefined, incoming: boolean): readonly Relationship[] {
  if (personId === undefined) {
    return []
  }
  return FIXTURE_RELATIONSHIPS.filter((relationship) =>
    incoming ? relationship.toPersonId === personId : relationship.fromPersonId === personId,
  )
}

function notFound(entity: string): Result<never> {
  return err('NOT_FOUND', `this ${entity} does not exist`)
}

function asString(value: unknown): string | undefined {
  return typeof value === 'string' && value !== '' ? value : undefined
}

function asNumber(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined
}

export type { Household, Person, Relationship, Term }
