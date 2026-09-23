import { readFile } from 'node:fs/promises'
import { runInNewContext } from 'node:vm'
import { test } from 'node:test'
import assert from 'node:assert/strict'
import ts from 'typescript'

const source = await readFile(new URL('../src/app/theme.ts', import.meta.url), 'utf8')
const { outputText } = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS } })
const html = await readFile(new URL('../index.html', import.meta.url), 'utf8')
const initialScript = html.match(/<script>([\s\S]*?)<\/script>/)[1]

function readThemes(values = {}, blocked = false, path = '/') {
  const localStorage = { getItem(key) { if (blocked) throw new Error('Storage unavailable'); return values[key] ?? null } }
  const context = { exports: {}, localStorage }
  runInNewContext(outputText, context)
  const document = { documentElement: { dataset: {} } }
  runInNewContext(initialScript, { localStorage, document, location: { pathname: path } })
  return { app: context.exports.savedTheme(), initial: document.documentElement.dataset.theme }
}

test('new browsers start in club before first paint and in the app', () => {
  assert.deepEqual(readThemes(), { app: 'club', initial: 'club' })
})
test('existing browsers receive club regardless of their legacy preference', () => {
  for (const theme of ['arena', 'cherry', 'light', 'club']) {
    assert.deepEqual(readThemes({ 'teamtime-theme': theme }), { app: 'club', initial: 'club' })
  }
})
test('choices made after the update survive reload, including arena', () => {
  for (const theme of ['arena', 'cherry', 'light', 'club']) {
    assert.deepEqual(readThemes({ 'teamtime-theme-v2': theme }), { app: theme, initial: theme })
  }
})
test('invalid or unavailable storage falls back to club in both entry paths', () => {
  assert.deepEqual(readThemes({ 'teamtime-theme-v2': 'unknown' }), { app: 'club', initial: 'club' })
  assert.deepEqual(readThemes({}, true), { app: 'club', initial: 'club' })
})
test('prepaint theme does not override the independent design lab', () => {
  assert.equal(readThemes({}, false, '/design-lab').initial, undefined)
})
