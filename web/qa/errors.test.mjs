import { readFile } from 'node:fs/promises'
import { test } from 'node:test'
import assert from 'node:assert/strict'
import ts from 'typescript'

const source = await readFile(new URL('../src/app/errors.ts', import.meta.url), 'utf8')
const { outputText } = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.ESNext } })
const { readableError } = await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)

test('both bot configuration errors explain how to resolve the problem', () => {
  const expected = 'Telegram-бот не подключён. Публикация недоступна — обратитесь к администратору сервиса.'
  assert.equal(readableError('bot is not configured', 400), expected)
  assert.equal(readableError('manual controls are unavailable: bot is not configured', 400), expected)
  assert.equal(readableError(expected), expected)
})

test('network failures and authorization errors are actionable in Russian', () => {
  assert.match(readableError(new TypeError('Failed to fetch')), /Проверьте подключение/)
  assert.match(readableError('unauthorized', 401), /Войдите/)
  assert.match(readableError('forbidden', 403), /нет прав/)
  assert.match(readableError('context deadline exceeded'), /не ответил вовремя/)
  assert.match(readableError('unknown', 429), /Подождите/)
})

test('existing Russian validation and partial-save results survive translation', () => {
  for (const message of ['Название события обязательно', 'Сохранено взносов: 1. Не удалось сохранить данные. Проверьте заполненные поля и повторите попытку.']) {
    assert.equal(readableError(message), message)
  }
})

test('unknown diagnostics and malformed error payloads never appear verbatim', () => {
  for (const message of ['unexpected EOF', 'pq: relation secret_table does not exist', '<html>Bad gateway</html>', { error:'internal detail' }, undefined]) {
    const result = readableError(message, 500)
    assert.match(result, /Сервис временно недоступен/)
    assert.doesNotMatch(result, /secret_table|EOF|html|internal/)
  }
  assert.doesNotMatch(readableError('Ошибка SQLSTATE 23505 INSERT INTO secrets'), /SQLSTATE|secrets/)
})

test('domain errors retain useful instructions instead of raw backend wording', () => {
  assert.match(readableError('no poll template is bound to event', 400), /привяжите шаблон/)
  assert.match(readableError('template requires at least 2 options', 400), /два варианта/)
  assert.match(readableError('event publications are disabled', 400), /Публикации.*выключены/)
})
