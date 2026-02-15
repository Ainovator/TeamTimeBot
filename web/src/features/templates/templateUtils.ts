export type TemplateFormState = {
  name: string
  question: string
  options: string[]
  counted: boolean[]
  weights: number[]
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
    weights: [...state.weights],
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
  while (next.weights.length < next.options.length) {
    next.weights.push(1)
  }
  if (next.weights.length > next.options.length) {
    next.weights = next.weights.slice(0, next.options.length)
  }
  return next
}

export function buildTemplatePayload(
  options: string[],
  countedFlags: boolean[],
  weights: number[],
): { options: string[]; countedOptions: number[]; optionWeights: number[] } {
  const cleanedOptions: string[] = []
  const countedOptions: number[] = []
  const optionWeights: number[] = []
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
    const w = Number.isFinite(weights[i]) ? weights[i] : 1
    optionWeights.push(w > 0 ? Math.floor(w) : 1)
  }
  return { options: cleanedOptions, countedOptions, optionWeights }
}
