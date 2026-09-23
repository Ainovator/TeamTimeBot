import { readFile } from 'node:fs/promises'
import { runInNewContext } from 'node:vm'
import { test } from 'node:test'
import assert from 'node:assert/strict'
import ts from 'typescript'

const html = await readFile(new URL('../index.html', import.meta.url), 'utf8')
const script = [...html.matchAll(/<script>([\s\S]*?)<\/script>/g)][1][1]
function boot() {
  const nodes = { 'boot-status': { textContent: 'Загружаем TeamTime…' }, 'boot-retry': { hidden: true } }
  let timeout, onError
  runInNewContext(script, { document: { getElementById: id => nodes[id] }, window: {
    setTimeout: callback => { timeout = callback },
    addEventListener: (type, callback) => { if (type === 'error') onError = callback },
  } })
  return { nodes, timeout, onError }
}
test('missing entry script shows recovery without loading React', () => {
  const state = boot()
  state.onError({ target: { tagName: 'SCRIPT' } })
  assert.equal(state.nodes['boot-retry'].hidden, false)
  assert.match(state.nodes['boot-status'].textContent, /Не удалось/)
})
test('a stalled initial load exposes recovery after the timeout', () => {
  const state = boot()
  state.timeout()
  assert.equal(state.nodes['boot-retry'].hidden, false)
})
test('the boot timeout does not interfere after React has mounted', () => {
  const state = boot()
  delete state.nodes['boot-status']
  delete state.nodes['boot-retry']
  assert.doesNotThrow(() => state.timeout())
})

const mediaSource = await readFile(new URL('../src/app/mediaQuery.ts', import.meta.url), 'utf8')
const { outputText } = ts.transpileModule(mediaSource, { compilerOptions: { module: ts.ModuleKind.CommonJS } })
for (const legacy of [false, true]) {
  test(`media queries work and unsubscribe with ${legacy ? 'legacy Safari' : 'modern'} APIs`, () => {
    let listener, removed
    const query = { matches: true }
    if (legacy) {
      query.addListener = fn => { listener = fn }
      query.removeListener = fn => { removed = fn }
    } else {
      query.addEventListener = (name, fn) => { assert.equal(name, 'change'); listener = fn }
      query.removeEventListener = (name, fn) => { assert.equal(name, 'change'); removed = fn }
    }
    const context = { exports: {}, window: { matchMedia: () => query } }
    runInNewContext(outputText, context)
    const states = []
    const stop = context.exports.watchMediaQuery('(pointer: coarse)', value => states.push(value))
    query.matches = false
    listener()
    stop()
    assert.deepEqual(states, [true, false])
    assert.equal(removed, listener)
  })
}
