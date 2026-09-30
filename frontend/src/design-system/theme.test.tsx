import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ThemeProvider, useTheme } from '@/app/providers/ThemeProvider'
import { DEFAULT_MODE, DEFAULT_SKIN } from '@/design-system/skins'
import { SkinSwitcher } from '@/app/shell/SkinSwitcher'

/**
 * The skin system is a promise to the user: the same components must render
 * correctly under every shipped skin and both modes. These tests exercise the
 * switching contract itself, not the colours.
 */
describe('theme', () => {
  beforeEach(() => {
    globalThis.localStorage.clear()
    document.documentElement.removeAttribute('data-skin')
    document.documentElement.removeAttribute('data-mode')
  })

  function Probe() {
    const { skin, mode, setSkin } = useTheme()
    return (
      <div>
        <span data-testid="skin">{skin}</span>
        <span data-testid="mode">{mode}</span>
        <button type="button" onClick={() => setSkin('latte')}>
          latte
        </button>
      </div>
    )
  }

  it('applies the default skin and mode to the document', () => {
    render(
      <ThemeProvider>
        <Probe />
      </ThemeProvider>,
    )

    expect(screen.getByTestId('skin')).toHaveTextContent(DEFAULT_SKIN)
    expect(screen.getByTestId('mode')).toHaveTextContent(DEFAULT_MODE)
    expect(document.documentElement.dataset.skin).toBe(DEFAULT_SKIN)
    expect(document.documentElement.dataset.mode).toBe(DEFAULT_MODE)
  })

  it('restores a stored preference', () => {
    globalThis.localStorage.setItem('edupilot.skin', 'latte')
    globalThis.localStorage.setItem('edupilot.mode', 'dark')

    render(
      <ThemeProvider>
        <Probe />
      </ThemeProvider>,
    )

    expect(screen.getByTestId('skin')).toHaveTextContent('latte')
    expect(document.documentElement.dataset.mode).toBe('dark')
  })

  it('persists a change so it survives a restart', async () => {
    const user = userEvent.setup()

    render(
      <ThemeProvider>
        <Probe />
        <SkinSwitcher />
      </ThemeProvider>,
    )

    await user.click(screen.getByRole('button', { name: 'latte' }))

    expect(screen.getByTestId('skin')).toHaveTextContent('latte')
    expect(globalThis.localStorage.getItem('edupilot.skin')).toBe('latte')
    expect(document.documentElement.dataset.skin).toBe('latte')
  })

  it('offers every shipped skin in the switcher', () => {
    render(
      <ThemeProvider>
        <SkinSwitcher />
      </ThemeProvider>,
    )

    const select = screen.getByLabelText('Appearance')
    const options = within(select).getAllByRole('option')
    expect(options.map((option) => option.getAttribute('value'))).toEqual(['navy', 'latte'])
  })

  it('toggles between light and dark', async () => {
    const user = userEvent.setup()

    render(
      <ThemeProvider>
        <SkinSwitcher />
      </ThemeProvider>,
    )

    await user.click(screen.getByRole('button', { name: /dark appearance/i }))

    expect(document.documentElement.dataset.mode).toBe('dark')
    expect(globalThis.localStorage.getItem('edupilot.mode')).toBe('dark')
  })
})
