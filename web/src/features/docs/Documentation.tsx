import { useEffect, useRef, useState } from 'react'
import { Icon } from '../../components/Icon'
import { Select } from '../../components/Select'
import './documentation.css'

const command = '/setgroup'
const chapters = [
  { id: 'docs-connect-group', title: 'Подключение группы' },
  { id: 'docs-registration', title: 'Регистрация игроков' },
  { id: 'docs-poll-template', title: 'Шаблон голосования' },
  { id: 'docs-event-template', title: 'Шаблон события' },
]

export function Documentation() {
  const [copyStatus, setCopyStatus] = useState<'idle' | 'copied' | 'failed'>('idle')
  const [copying, setCopying] = useState(false)
  const mounted = useRef(false)
  const resetTimer = useRef<ReturnType<typeof setTimeout>>()
  const guideRef = useRef<HTMLDivElement>(null)
  const navigationRef = useRef<HTMLElement>(null)

  useEffect(() => {
    const guide = guideRef.current
    const navigation = navigationRef.current
    if (!guide || !navigation) return
    const updateNavigationHeight = () => {
      guide.style.setProperty('--guide-navigation-height', `${navigation.getBoundingClientRect().height}px`)
    }
    updateNavigationHeight()
    const observer = new ResizeObserver(updateNavigationHeight)
    observer.observe(navigation)
    return () => observer.disconnect()
  }, [])

  function openChapter(id: string) {
    document.getElementById(id)?.scrollIntoView({ block: 'start' })
  }

  useEffect(() => {
    mounted.current = true
    return () => {
      mounted.current = false
      clearTimeout(resetTimer.current)
    }
  }, [])

  async function copyCommand() {
    if (copying) return
    clearTimeout(resetTimer.current)
    setCopying(true)
    try {
      await navigator.clipboard.writeText(command)
      if (!mounted.current) return
      setCopyStatus('copied')
      resetTimer.current = setTimeout(() => setCopyStatus('idle'), 3000)
    } catch {
      if (mounted.current) setCopyStatus('failed')
    } finally {
      if (mounted.current) setCopying(false)
    }
  }

  return (
    <div className="documentation" ref={guideRef}>
    <nav className="guide-chapter-nav" aria-label="Шаги настройки TeamTime" ref={navigationRef}>
      <div className="guide-desktop-nav">
        {chapters.map((chapter, index) => (
          <button type="button" key={chapter.id} onClick={() => openChapter(chapter.id)}>
            <span>Шаг {index}</span>{chapter.title}
          </button>
        ))}
      </div>
      <label className="guide-mobile-nav">
        <span>Шаги настройки</span>
        <Select aria-label="Перейти к шагу" value="" onChange={event => openChapter(event.target.value)}>
          <option value="" disabled>Перейти к шагу…</option>
          {chapters.map((chapter, index) => (
            <option key={chapter.id} value={chapter.id}>Шаг {index} · {chapter.title}</option>
          ))}
        </Select>
      </label>
    </nav>
    <div className="teamtime-guide">
      <div className="guide-chapters">
      <article className="guide-article" aria-labelledby="guide-title" id="docs-connect-group">
        <header className="guide-intro">
          <div className="guide-meta">
            <span className="guide-step-label">Шаг 0</span>
            <span>Начало работы</span>
          </div>
          <h2 id="guide-title">Подключите Telegram-группу</h2>
          <p>
            TeamTime публикует голосования и сообщения в группе вашей команды.
            Для начала добавьте туда бота и подключите группу.
          </p>
        </header>

        <section className="guide-preparation" aria-labelledby="guide-preparation-title">
          <h3 id="guide-preparation-title">Перед началом</h3>
          <p>Вам понадобятся:</p>
          <ul>
            <li><Icon name="check" size={16}/><span>Группа вашей команды в Telegram</span></li>
            <li><Icon name="check" size={16}/><span>Права администратора этой группы</span></li>
          </ul>
        </section>

        <ol className="guide-steps" aria-label="Подключение Telegram-группы">
          <li>
            <span className="guide-step-number" aria-hidden="true">1</span>
            <div className="guide-step-content">
              <h3>Добавьте бота в группу</h3>
              <p>
                Откройте профиль бота TeamTime в Telegram, выберите <strong>«Добавить в группу»</strong>
                {' '}и укажите группу вашей команды.
              </p>
              <div className="guide-location"><Icon name="users" size={16}/>В Telegram</div>
            </div>
          </li>
          <li>
            <span className="guide-step-number" aria-hidden="true">2</span>
            <div className="guide-step-content">
              <h3>Подключите группу</h3>
              <p>Отправьте в чате группы команду:</p>
              <div className="guide-command">
                <code>{command}</code>
                <button type="button" className="guide-copy" onClick={() => void copyCommand()}
                  disabled={copying} aria-label="Скопировать команду /setgroup">
                  {copyStatus === 'copied' ? <Icon name="check" size={17}/> : (
                    <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor"
                      strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                      <rect x="8" y="8" width="12" height="13" rx="2"/>
                      <path d="M16 8V5a2 2 0 0 0-2-2H5a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h3"/>
                    </svg>
                  )}
                  <span>{copyStatus === 'copied' ? 'Скопировано' : 'Копировать'}</span>
                </button>
              </div>
              <p className="guide-copy-status" role="status">
                {copyStatus === 'failed'
                  ? 'Не удалось скопировать. Выделите команду и скопируйте её вручную.'
                  : copyStatus === 'copied' ? 'Команда скопирована. Вставьте её в чат группы.' : ''}
              </p>
              <p className="guide-permission">
                Команду должен отправить <strong>администратор группы</strong>.
              </p>
            </div>
          </li>
          <li>
            <span className="guide-step-number" aria-hidden="true">3</span>
            <div className="guide-step-content">
              <h3>Дождитесь подтверждения</h3>
              <p>
                Бот ответит, что группа сохранена, а администраторы синхронизированы.
                В сообщении будут указаны название группы и её часовой пояс.
              </p>
              <figure className="guide-example">
                <figcaption>Пример ответа бота</figcaption>
                <div className="guide-message">
                  <div className="guide-message-author">
                    <span className="guide-bot-avatar" aria-hidden="true"><Icon name="ball" size={21}/></span>
                    <div><strong>TeamTime</strong><span>Бот</span></div>
                  </div>
                  <blockquote>
                    Группа сохранена: Ваша команда (Europe/Moscow). Админы синхронизированы.
                  </blockquote>
                </div>
              </figure>
            </div>
          </li>
        </ol>

        <footer className="guide-result">
          <span className="guide-result-icon"><Icon name="check" size={20}/></span>
          <div>
            <h3>После подтверждения</h3>
            <p>Группа подключена к TeamTime — можно переходить к настройке в приложении.</p>
          </div>
        </footer>
      </article>

      <article className="guide-article" aria-labelledby="guide-registration-title" id="docs-registration">
        <header className="guide-intro">
          <div className="guide-meta">
            <span className="guide-step-label">Шаг 1</span>
            <span>Начало работы</span>
          </div>
          <h2 id="guide-registration-title">Откройте регистрацию игроков</h2>
          <p>
            Организация уже подключена к TeamTime. Теперь опубликуйте регистрационный опрос
            для участников вашей команды.
          </p>
        </header>

        <ol className="guide-steps" aria-label="Публикация регистрации игроков">
          <li>
            <span className="guide-step-number" aria-hidden="true">1</span>
            <div className="guide-step-content">
              <h3>Войдите в TeamTime</h3>
              <p>Войдите в приложение через Telegram.</p>
            </div>
          </li>
          <li>
            <span className="guide-step-number" aria-hidden="true">2</span>
            <div className="guide-step-content">
              <h3>Выберите организацию</h3>
              <p>В списке <strong>«Ваша команда»</strong> в верхней части страницы выберите организацию вашей команды.</p>
            </div>
          </li>
          <li>
            <span className="guide-step-number" aria-hidden="true">3</span>
            <div className="guide-step-content">
              <h3>Откройте раздел «Организация»</h3>
              <p>В верхней части приложения нажмите <strong>«Организация»</strong>.</p>
              <div className="guide-location"><Icon name="organization" size={16}/>В TeamTime</div>
            </div>
          </li>
          <li>
            <span className="guide-step-number" aria-hidden="true">4</span>
            <div className="guide-step-content">
              <h3>Опубликуйте регистрацию</h3>
              <p>
                В блоке <strong>«Регистрация игроков»</strong> нажмите <strong>«Опубликовать регистрацию»</strong>.
              </p>
            </div>
          </li>
        </ol>

        <footer className="guide-result">
          <span className="guide-result-icon"><Icon name="check" size={20}/></span>
          <div>
            <h3>После публикации</h3>
            <p>В Telegram-группе появится регистрационный опрос для участников команды.</p>
          </div>
        </footer>
      </article>

      <article className="guide-article" aria-labelledby="guide-poll-template-title" id="docs-poll-template">
        <header className="guide-intro">
          <div className="guide-meta">
            <span className="guide-step-label">Шаг 2</span>
            <span>Начало работы</span>
          </div>
          <h2 id="guide-poll-template-title">Создайте шаблон голосования</h2>
          <p>Шаблон определяет вопрос и варианты ответов для записи на тренировку.</p>
        </header>

        <ol className="guide-steps" aria-label="Создание шаблона голосования">
          <li>
            <span className="guide-step-number" aria-hidden="true">1</span>
            <div className="guide-step-content">
              <h3>Откройте создание шаблона</h3>
              <p>
                В верхней части приложения нажмите <strong>«Организация»</strong>.
                В блоке <strong>«Шаблоны»</strong> выберите <strong>«Шаблоны голосований»</strong>,
                затем нажмите <strong>«Создать шаблон»</strong>.
              </p>
              <div className="guide-location"><Icon name="template" size={16}/>В TeamTime</div>
            </div>
          </li>
          <li>
            <span className="guide-step-number" aria-hidden="true">2</span>
            <div className="guide-step-content">
              <h3>Укажите название и вопрос</h3>
              <p>
                В поле <strong>«Название»</strong> введите, например, «Запись на тренировку».
                В поле <strong>«Вопрос»</strong> — «Кто придёт на тренировку?».
              </p>
            </div>
          </li>
          <li>
            <span className="guide-step-number" aria-hidden="true">3</span>
            <div className="guide-step-content">
              <h3>Добавьте варианты ответов</h3>
              <p>
                В блоке <strong>«Варианты»</strong> укажите ответы <strong>«Приду»</strong> и <strong>«Не смогу»</strong>.
                Если игроки могут приходить с гостями, нажмите <strong>«+»</strong> под списком
                и добавьте <strong>«Со мной +1»</strong> и <strong>«Со мной +2»</strong>.
              </p>
            </div>
          </li>
          <li>
            <span className="guide-step-number" aria-hidden="true">4</span>
            <div className="guide-step-content">
              <h3>Настройте «Учёт» и коэффициент K</h3>
              <p>
                <strong>«Учёт»</strong> включает ответ в подсчёт мест на тренировке.
                Включите его у ответа «Приду» и вариантов с гостями. У ответа «Не смогу» оставьте выключенным.
              </p>
              <p className="guide-paragraph">
                <strong>K — сколько мест добавляет выбранный ответ с включённым «Учётом».</strong>
                {' '}K = 1 добавляет одно место, K = 2 — два. Введите целое число от 1.
                Если «Учёт» выключен, ответ не добавляет мест независимо от K.
              </p>

              <table className="guide-options-table">
                <caption>Пример настройки ответов</caption>
                <thead>
                  <tr><th scope="col">Вариант</th><th scope="col">K</th><th scope="col">Учёт</th></tr>
                </thead>
                <tbody>
                  <tr><th scope="row">Приду</th><td>1</td><td>Включён</td></tr>
                  <tr><th scope="row">Со мной +1</th><td>1</td><td>Включён</td></tr>
                  <tr><th scope="row">Со мной +2</th><td>2</td><td>Включён</td></tr>
                  <tr><th scope="row">Не смогу</th><td>1</td><td>Выключен</td></tr>
                </tbody>
              </table>

              <h4 className="guide-subheading">Как голосовать с гостями</h4>
              <p>В опросе можно выбрать несколько ответов. Коэффициенты выбранных ответов с «Учётом» складываются:</p>
              <ul className="guide-examples">
                <li><strong>Приду один:</strong> «Приду» → 1 место.</li>
                <li><strong>Приду с одним гостем:</strong> «Приду» + «Со мной +1» → 1 + 1 = 2 места.</li>
                <li><strong>Приду с двумя гостями:</strong> «Приду» + «Со мной +2» → 1 + 2 = 3 места.</li>
              </ul>
              <p>
                Для двух гостей выберите «Со мной +2»; «Со мной +1» при этом не отмечайте,
                иначе его место тоже добавится. Если не идёте, оставьте только «Не смогу»:
                этот ответ сам по себе не отменяет другие выбранные варианты.
              </p>

              <div className="guide-option-note">
                <h4>Если нужен один ответ «Я и друг»</h4>
                <p>
                  Задайте ему <strong>K = 2</strong> и включите <strong>«Учёт»</strong>.
                  Игрок выбирает только «Я и друг», без дополнительного «Приду»: этот ответ уже включает обоих.
                  Количество мест задаётся полем K — число в названии ответа автоматически не распознаётся.
                </p>
              </div>
            </div>
          </li>
          <li>
            <span className="guide-step-number" aria-hidden="true">5</span>
            <div className="guide-step-content">
              <h3>Сохраните шаблон</h3>
              <p>Нажмите <strong>«Сохранить шаблон»</strong>.</p>
            </div>
          </li>
        </ol>

        <footer className="guide-result">
          <span className="guide-result-icon"><Icon name="check" size={20}/></span>
          <div>
            <h3>После сохранения</h3>
            <p>Шаблон голосования готов — следующим шагом привяжем его к шаблону события.</p>
          </div>
        </footer>
      </article>

      <article className="guide-article" aria-labelledby="guide-event-template-title" id="docs-event-template">
        <header className="guide-intro">
          <div className="guide-meta">
            <span className="guide-step-label">Шаг 3</span>
            <span>Начало работы</span>
          </div>
          <h2 id="guide-event-template-title">Создайте шаблон события</h2>
          <p>
            Шаблон события объединяет расписание тренировки, количество мест и голосование для записи.
            Подключите к нему шаблон голосования, который создали на предыдущем шаге.
          </p>
        </header>

        <ol className="guide-steps" aria-label="Создание шаблона события">
          <li>
            <span className="guide-step-number" aria-hidden="true">1</span>
            <div className="guide-step-content">
              <h3>Откройте создание шаблона</h3>
              <p>
                В верхней части приложения нажмите <strong>«Организация»</strong>.
                В блоке <strong>«Шаблоны»</strong> выберите <strong>«Шаблоны событий»</strong>,
                затем нажмите <strong>«Создать шаблон»</strong>.
              </p>
              <div className="guide-location"><Icon name="template" size={16}/>В TeamTime</div>
            </div>
          </li>
          <li>
            <span className="guide-step-number" aria-hidden="true">2</span>
            <div className="guide-step-content">
              <h3>Укажите название и расписание</h3>
              <p>
                В поле <strong>«Название»</strong> введите, например, «Тренировка по средам».
                В поле <strong>«Тип события»</strong> выберите <strong>«Тренировка»</strong>
                {' '}или <strong>«Мероприятие»</strong>.
              </p>
              <p className="guide-paragraph">
                Укажите <strong>«День недели»</strong> и время в полях <strong>«Начало»</strong>
                {' '}и <strong>«Конец»</strong>. Например: среда, с 20:00 до 22:00.
              </p>
            </div>
          </li>
          <li>
            <span className="guide-step-number" aria-hidden="true">3</span>
            <div className="guide-step-content">
              <h3>Задайте количество мест и стоимость</h3>
              <p>
                В поле <strong>«Максимум мест»</strong> укажите, сколько участников может прийти,
                включая гостей. Например, 12.
              </p>
              <p className="guide-paragraph">
                Если у события есть общая стоимость, укажите её в поле <strong>«Стоимость»</strong>.
                Например, 4000 за аренду зала. Это сумма за всё событие.
                Если стоимость пока неизвестна, поле можно оставить пустым.
              </p>
            </div>
          </li>
          <li>
            <span className="guide-step-number" aria-hidden="true">4</span>
            <div className="guide-step-content">
              <h3>Привяжите голосование</h3>
              <p>
                В поле <strong>«Шаблон опроса»</strong> выберите голосование из предыдущего шага —
                например, <strong>«Запись на тренировку»</strong>.
                Его вопрос и варианты ответов будут использоваться для записи на событие.
              </p>
            </div>
          </li>
          <li>
            <span className="guide-step-number" aria-hidden="true">5</span>
            <div className="guide-step-content">
              <h3>Выберите, когда открывать запись</h3>
              <p>
                В поле <strong>«День публикации опроса»</strong> выберите день,
                а в поле <strong>«Публикация опроса»</strong> укажите время.
                Это момент, когда голосование должно появиться в Telegram-группе.
              </p>
              <div className="guide-option-note">
                <h4>Пример: запись открывается заранее</h4>
                <p>
                  Тренировка проходит <strong>в среду с 20:00 до 22:00</strong>.
                  Для публикации опроса выберите <strong>понедельник, 12:00</strong> —
                  участники смогут записаться до начала тренировки.
                </p>
              </div>
            </div>
          </li>
          <li>
            <span className="guide-step-number" aria-hidden="true">6</span>
            <div className="guide-step-content">
              <h3>Сохраните шаблон события</h3>
              <p>Проверьте заполненные поля и нажмите <strong>«Создать шаблон»</strong>.</p>
            </div>
          </li>
        </ol>

        <footer className="guide-result">
          <span className="guide-result-icon"><Icon name="check" size={20}/></span>
          <div>
            <h3>После сохранения</h3>
            <p>В списке шаблонов событий появится ваш шаблон с расписанием, количеством мест и привязанным голосованием.</p>
          </div>
        </footer>
      </article>
      </div>

    </div>
    </div>
  )
}
