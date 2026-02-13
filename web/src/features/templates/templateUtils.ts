export type TemplateFormState = {
  name: string
  question: string
  options: string[]
  counted: boolean[]
}

export function ensureAtLeastTwoOptions(options: string[]): string[] {
  const prepared = [...options]
  while (prepared.length < 2) {
    prepared.push('')
  }
  return prepared
}

export function ensureAtLeastTwoTemplateState(state: TemplateFormState): TemplateFormState {
  const next = {
    ...state,
    options: [...state.options],
    counted: [...state.counted],
  }
  while (next.options.length < 2) {
    next.options.push('')
  }
  while (next.counted.length < next.options.length) {
    next.counted.push(false)
  }
  if (next.counted.length > next.options.length) {
    next.counted = next.counted.slice(0, next.options.length)
  }
  return next
}

export function buildTemplatePayload(options: string[], countedFlags: boolean[]): { options: string[]; countedOptions: number[] } {
  const cleanedOptions: string[] = []
  const countedOptions: number[] = []
  for (let i = 0; i < options.length; i += 1) {
    const value = options[i].trim()
    if (!value) {
      continue
    }
    const nextIndex = cleanedOptions.length
    cleanedOptions.push(value)
    if (countedFlags[i]) {
      countedOptions.push(nextIndex)
    }
  }
  return { options: cleanedOptions, countedOptions }
}
