export type ProductTheme = 'arena' | 'club' | 'cherry' | 'light'

// A new preference key gives existing browsers the new default once.
// Explicit choices made after this update still persist across visits.
export const themeStorageKey = 'teamtime-theme-v2'
export const defaultTheme: ProductTheme = 'club'

export function savedTheme(): ProductTheme {
  try {
    const stored = localStorage.getItem(themeStorageKey)
    return stored === 'arena' || stored === 'light' || stored === 'cherry' || stored === 'club' ? stored : defaultTheme
  } catch { return defaultTheme }
}
