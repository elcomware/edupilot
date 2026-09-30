export interface NavItem {
  readonly key: string
  readonly path: string
}

export interface NavSection {
  readonly key: string
  readonly path: string
  readonly items: readonly NavItem[]
}
