const fallback = 'Не удалось выполнить действие. Попробуйте ещё раз.'

/** Translate known API failures; never expose unknown backend diagnostics to the UI. */
export function readableError(error: unknown, status?: number): string {
  const message = (error instanceof Error ? error.message : typeof error === 'string' ? error : '').trim()
  const rules: [RegExp, string][] = [
    [/Опрос опубликован, но не удалось отправить упоминания/i, 'Опрос опубликован, но упоминания не отправлены. Проверьте Telegram-группу; повторно публиковать опрос не нужно.'],
    [/mention recipients must be active members/i, 'В списке упоминаний есть игроки, которые больше не состоят в организации. Уберите их из списка и сохраните шаблон.'],
    [/mention user ID must be positive/i, 'Не удалось определить игрока для упоминания. Обновите список и выберите его снова.'],
    [/bot is not configured/i, 'Telegram-бот не подключён. Публикация недоступна — обратитесь к администратору сервиса.'],
    [/(?:telegram )?auth is not configured/i, 'Вход через Telegram ещё не настроен. Обратитесь к администратору сервиса.'],
    [/telegram auth verification failed|invalid telegram (?:auth payload|hash)|telegram auth payload expired/i, 'Не удалось подтвердить вход через Telegram. Войдите ещё раз.'],
    [/unauthorized|session expired|invalid session/i, 'Сеанс завершён. Войдите в приложение ещё раз.'],
    [/forbidden|permission denied|access denied/i, 'У вас нет прав на это действие. Обратитесь к администратору команды.'],
    [/failed to fetch|networkerror|network request failed|load failed|fetch failed/i, 'Не удалось связаться с сервером. Проверьте подключение к интернету и попробуйте ещё раз.'],
    [/timeout|timed out|deadline exceeded/i, 'Сервер не ответил вовремя. Попробуйте ещё раз чуть позже.'],
    [/too many requests|too many attempts|retry after/i, 'Слишком много запросов. Подождите немного и повторите попытку.'],
    [/bot was kicked|bot is not a member|chat not found/i, 'Бот не имеет доступа к Telegram-группе. Попросите администратора проверить его подключение.'],
    [/not enough rights|chat_write_forbidden|need administrator rights/i, 'У бота недостаточно прав в Telegram-группе. Попросите администратора разрешить публикацию сообщений и опросов.'],
    [/poll has already been closed|poll is closed/i, 'Это голосование уже закрыто. Обновите данные команды.'],
    [/message to delete not found|message to edit not found/i, 'Сообщение уже удалено из Telegram. Обновите данные команды.'],
    [/telegram did not return poll metadata/i, 'Не удалось подтвердить публикацию опроса. Проверьте Telegram-группу перед повторной попыткой.'],
    [/group not found/i, 'Организация не найдена. Обновите страницу или подключите группу через бота.'],
    [/event publications are disabled/i, 'Публикации для этого события выключены. Включите их в шаблоне события.'],
    [/announcement text is empty/i, 'Добавьте текст анонса перед публикацией.'],
    [/no poll template is bound to event/i, 'Сначала привяжите шаблон голосования к событию.'],
    [/need at least 2 options|template requires at least 2 options/i, 'Добавьте в голосование хотя бы два варианта ответа.'],
    [/template has no options marked with accounting flag/i, 'Отметьте хотя бы один вариант ответа для учёта участников в шаблоне голосования.'],
    [/no published poll found/i, 'Сначала опубликуйте голосование для этого события.'],
    [/no players to publish/i, 'Нет игроков для публикации состава. Добавьте участников и распределите их по командам.'],
    [/no sets to publish/i, 'Сначала сохраните результаты партий.'],
    [/no debtors to publish/i, 'Неоплаченных взносов нет — публиковать напоминание не требуется.'],
    [/nothing to publish/i, 'Пока нет данных для публикации.'],
    [/billing not found/i, 'Расчёт взносов ещё не создан для этой встречи.'],
    [/team1 and team2 are required/i, 'Выберите обе команды.'],
    [/userTelegramID is required|invalid (?:related )?user id/i, 'Выберите игрока из списка команды.'],
    [/question is required/i, 'Введите вопрос для голосования.'],
    [/invalid time format/i, 'Укажите время в формате ЧЧ:ММ, например 19:30.'],
    [/invalid template name/i, 'Укажите корректное название шаблона.'],
    [/invalid (?:chat|instance|post|event|schedule) id/i, 'Не удалось определить выбранный объект. Обновите страницу и выберите его заново.'],
    [/invalid relation type/i, 'Выберите тип связи между игроками.'],
    [/event.*not found|instance.*not found/i, 'Событие не найдено. Возможно, оно удалено — обновите список.'],
    [/template.*not found/i, 'Шаблон не найден. Обновите список и выберите другой.'],
    [/member.*not found|user.*not found|player.*not found/i, 'Игрок не найден в команде. Обновите список участников.'],
    [/duplicate key|already exists/i, 'Такая запись уже существует. Проверьте данные перед сохранением.'],
  ]
  for (const [pattern, translation] of rules) if (pattern.test(message)) return translation
  // Keep existing domain validation messages, including useful partial-save counts.
  if (/[А-Яа-яЁё]/.test(message) && !/SQLSTATE|SELECT\s|INSERT\s|UPDATE\s|DELETE\s|panic:|goroutine|<html|<!doctype/i.test(message)) return message
  const code = status ?? Number(message.match(/^HTTP (\d{3})$/)?.[1])
  if (code === 401) return 'Сеанс завершён. Войдите в приложение ещё раз.'
  if (code === 403) return 'У вас нет прав на это действие. Обратитесь к администратору команды.'
  if (code === 404) return 'Данные не найдены. Обновите страницу.'
  if (code === 409) return 'Данные уже изменились. Обновите страницу и повторите действие.'
  if (code === 429) return 'Слишком много запросов. Подождите немного и повторите попытку.'
  if (code >= 500) return 'Сервис временно недоступен. Попробуйте ещё раз чуть позже.'
  if (code === 400 || code === 422) return 'Не удалось сохранить данные. Проверьте заполненные поля и повторите попытку.'
  return fallback
}
