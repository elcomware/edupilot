import { describe, expect, it } from 'vitest'
import { formatDate, initials, relationshipLabel, roleSummary } from './labels'
import { setLocale } from '@/i18n'
import { createPreviewTransport } from '@/gateway/preview'
import type { Person, Page } from '@/gateway/types'

describe('formatDate', () => {
  it('reads a calendar date in UTC so it cannot shift a day', () => {
    // A bare YYYY-MM-DD read as local time would show the 28th west of
    // Greenwich, which is somebody's wrong birthday.
    expect(formatDate('2025-09-01')).toMatch(/2025/)
    expect(formatDate('2025-09-01')).toMatch(/1/)
    expect(formatDate('2025-09-01')).not.toMatch(/31/)
  })

  it('shows a dash for a date the school has not recorded', () => {
    expect(formatDate('')).toBe('—')
  })

  it('passes through anything it cannot parse rather than hiding it', () => {
    expect(formatDate('not a date')).toBe('not a date')
  })
})

describe('initials', () => {
  it('takes the first and last name', () => {
    expect(initials('Amina Njoya')).toBe('AN')
    expect(initials('Joseph Marie Mbarga')).toBe('JM')
  })

  it('still gives a tile for a person with one name only', () => {
    expect(initials('Prince')).toBe('P')
  })

  it('does not crash on an empty name', () => {
    expect(initials('   ')).toBe('?')
  })
})

describe('roleSummary', () => {
  it('says so plainly when a person holds no role this year', () => {
    setLocale('en')
    expect(roleSummary([])).toBe('No role this year')
  })

  it('lists every role a person holds, because one is never the whole story', () => {
    setLocale('en')
    expect(roleSummary([{ role: 'STUDENT' }, { role: 'GUARDIAN' }])).toBe('Student · Guardian')
  })
})

describe('relationshipLabel', () => {
  it('reads as a sentence, not as a wire code', () => {
    setLocale('en')
    expect(relationshipLabel('GUARDIAN_OF', 'MOTHER')).toBe('Mother of')
  })

  it('falls back to the type when no role was recorded', () => {
    setLocale('en')
    expect(relationshipLabel('EMERGENCY_CONTACT_OF', '')).toBe('Emergency contact for')
  })
})

describe('preview transport', () => {
  const transport = createPreviewTransport()

  async function call<T>(operation: string, params?: Record<string, unknown>): Promise<T> {
    const result = await transport.call<T>(operation, params)
    if (!result.ok) {
      throw new Error(result.error.message)
    }
    return result.value
  }

  it('paginates, and reports a total larger than the page', async () => {
    const page = await call<Page<Person>>('platform.people.App.ListPeople', { limit: 3, offset: 0 })
    expect(page.items).toHaveLength(3)
    // A total beyond the page is what a bursar paging through a school needs.
    expect(page.total).toBeGreaterThan(page.items.length)
  })

  it('searches by name', async () => {
    const page = await call<Page<Person>>('platform.people.App.ListPeople', { limit: 25, offset: 0, search: 'owona' })
    expect(page.items.length).toBeGreaterThan(0)
    for (const person of page.items) {
      expect(person.displayName.toLowerCase()).toContain('owona')
    }
  })

  it('returns an empty page rather than an error for a search that matches nobody', async () => {
    const page = await call<Page<Person>>('platform.people.App.ListPeople', { limit: 25, offset: 0, search: 'zzzz' })
    expect(page.items).toEqual([])
    expect(page.total).toBe(0)
  })

  it('includes roles on a person read individually', async () => {
    const person = await call<Person>('platform.people.App.GetPerson', { id: 'person-2' })
    // A role without its profile is the defect this guards against: the person
    // screen would show "Employee" with no job title beside it.
    expect(person.roles).toHaveLength(1)
    expect(person.roles[0]?.employee?.jobTitle).toBe('Mathematics Teacher')
  })

  it('reports a person that does not exist as not found', async () => {
    const result = await transport.call<Person>('platform.people.App.GetPerson', { id: 'nope' })
    expect(result.ok).toBe(false)
  })

  it('reads relationships from both ends', async () => {
    const outward = await call<readonly { toPersonId: string }[]>('platform.people.App.ListRelationships', {
      personId: 'person-2',
    })
    const inward = await call<readonly { fromPersonId: string }[]>('platform.people.App.ListRelationships', {
      personId: 'person-3',
      incoming: true,
    })
    expect(outward.some((edge) => edge.toPersonId === 'person-3')).toBe(true)
    expect(inward.some((edge) => edge.fromPersonId === 'person-2')).toBe(true)
  })
})
