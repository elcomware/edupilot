import { useTheme } from '@/app/providers/ThemeProvider'
import { SKINS, type Mode, type Skin } from '@/design-system/skins'
import { t } from '@/i18n'

const controlClass =
  'h-7 rounded-[var(--radius-control)] border border-line bg-surface px-2 text-xs text-ink hover:border-accent hover:text-accent'

/**
 * SkinSwitcher is shell chrome, not a component of the design system: it knows
 * about the shipped skins and about translations, which a generic control
 * should not.
 */
export function SkinSwitcher() {
  const { skin, mode, setSkin, toggleMode } = useTheme()

  return (
    <div className="flex items-center gap-2 print:hidden">
      <label className="sr-only" htmlFor="ep-skin">
        {t('theme.skin')}
      </label>
      <select
        id="ep-skin"
        className={controlClass}
        value={skin}
        onChange={(event) => setSkin(event.target.value as Skin)}
      >
        {SKINS.map((candidate) => (
          <option key={candidate} value={candidate}>
            {t(`theme.skins.${candidate}`)}
          </option>
        ))}
      </select>

      <button
        type="button"
        className={controlClass}
        onClick={toggleMode}
        aria-label={t(`theme.mode.${mode === 'dark' ? 'toLight' : 'toDark'}`)}
        title={t(`theme.mode.${mode === 'dark' ? 'toLight' : 'toDark'}`)}
      >
        {mode === 'dark' ? '☾' : '☀'}
      </button>
    </div>
  )
}

export type { Mode, Skin }
