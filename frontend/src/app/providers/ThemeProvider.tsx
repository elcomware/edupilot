import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import {
  DEFAULT_MODE,
  DEFAULT_SKIN,
  isMode,
  isSkin,
  type Mode,
  type Skin,
} from '@/design-system/skins'

/**
 * ThemeProvider owns the skin and the colour mode.
 *
 * Both are written to `data-skin` and `data-mode` on <html>, which is the only
 * coupling between a component and a skin: components keep using semantic
 * utilities such as `bg-surface` and inherit whatever the skin defines.
 *
 * The choice is stored per browser for now. When the settings module lands, the
 * same value is read from the user's account so a school's preference follows
 * them between machines.
 */
const SKIN_STORAGE_KEY = 'edupilot.skin'
const MODE_STORAGE_KEY = 'edupilot.mode'

export interface Theme {
  readonly skin: Skin
  readonly mode: Mode
  readonly setSkin: (skin: Skin) => void
  readonly toggleMode: () => void
}

const ThemeContext = createContext<Theme | null>(null)

function storedSkin(): Skin {
  const value = globalThis.localStorage?.getItem(SKIN_STORAGE_KEY) ?? null
  return isSkin(value) ? value : DEFAULT_SKIN
}

function storedMode(): Mode {
  const value = globalThis.localStorage?.getItem(MODE_STORAGE_KEY) ?? null
  return isMode(value) ? value : DEFAULT_MODE
}

function apply(skin: Skin, mode: Mode): void {
  const root = globalThis.document?.documentElement
  if (root === undefined) {
    return
  }
  root.dataset.skin = skin
  root.dataset.mode = mode
}

export interface ThemeProviderProps {
  readonly children: ReactNode
}

export function ThemeProvider({ children }: ThemeProviderProps) {
  const [skin, setSkinState] = useState<Skin>(storedSkin)
  const [mode, setMode] = useState<Mode>(storedMode)

  // The attributes are applied before the first paint of a repaint and on every
  // change, so there is never a flash of the wrong skin.
  useEffect(() => {
    apply(skin, mode)
  }, [skin, mode])

  const setSkin = useCallback((next: Skin) => {
    setSkinState(next)
    globalThis.localStorage?.setItem(SKIN_STORAGE_KEY, next)
  }, [])

  const toggleMode = useCallback(() => {
    setMode((current) => {
      const next: Mode = current === 'dark' ? 'light' : 'dark'
      globalThis.localStorage?.setItem(MODE_STORAGE_KEY, next)
      return next
    })
  }, [])

  const theme = useMemo<Theme>(
    () => ({ skin, mode, setSkin, toggleMode }),
    [skin, mode, setSkin, toggleMode],
  )

  return <ThemeContext.Provider value={theme}>{children}</ThemeContext.Provider>
}

export function useTheme(): Theme {
  const theme = useContext(ThemeContext)
  if (theme === null) {
    throw new Error('useTheme must be used inside a ThemeProvider')
  }
  return theme
}
