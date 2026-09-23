/** Safari versions before MediaQueryList became an EventTarget use addListener. */
export function watchMediaQuery(query: string, onChange: (matches: boolean) => void) {
  const media = window.matchMedia(query)
  const update = () => onChange(media.matches)
  update()
  if (typeof media.addEventListener === 'function') {
    media.addEventListener('change', update)
    return () => media.removeEventListener('change', update)
  }
  media.addListener(update)
  return () => media.removeListener(update)
}
