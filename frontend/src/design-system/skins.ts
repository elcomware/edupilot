/**
 * Skins are the shipped appearance variants. Adding one means adding a
 * `[data-skin='…']` block in theme.css and one entry here: no component knows
 * the name of a skin.
 */
export const SKINS = ['navy', 'latte'] as const

export type Skin = (typeof SKINS)[number]

/** Colour modes. A skin provides the hues; a mode decides light or dark. */
export const MODES = ['light', 'dark'] as const

export type Mode = (typeof MODES)[number]

export const DEFAULT_SKIN: Skin = 'navy'
export const DEFAULT_MODE: Mode = 'light'

export function isSkin(value: string | null): value is Skin {
  return value !== null && (SKINS as readonly string[]).includes(value)
}

export function isMode(value: string | null): value is Mode {
  return value !== null && (MODES as readonly string[]).includes(value)
}
