# Task Board: клиент-серверное приложение

Учебный проект для задания «подтяни Git, настрой GitHub и пойми механику клиентно-серверных приложений от идеи до запуска».

## 1. Что мы строим

Доску задач. Пользователь вводит задачу, браузер отправляет HTTP-запрос, Node.js проверяет данные и сохраняет их в `data/tasks.json`, затем возвращает JSON. Интерфейс показывает результат.

Это намеренно маленький проект: в нём видны фундаментальные части приложения без магии фреймворка.

## 2. Структура

```text
.
├── server.js          # HTTP-сервер, API, валидация, работа с JSON
├── public/
│   ├── index.html     # разметка клиентской части
│   ├── styles.css     # внешний вид
│   └── app.js         # fetch, события, отрисовка
├── test/server.test.js# автоматические проверки API
├── data/.gitkeep      # папка для локальных данных
├── package.json       # команды и описание проекта
└── .gitignore         # что не отправляем в GitHub
```

## 3. Как проходит один сценарий

1. Пользователь нажимает «Добавить».
2. `public/app.js` перехватывает отправку формы и вызывает `fetch('/api/tasks', { method: 'POST' })`.
3. Браузер отправляет HTTP-запрос на `localhost:3000` с JSON в body.
4. `server.js` находит маршрут `POST /api/tasks`, проверяет заголовок/данные и длину названия.
5. Сервер создаёт `id`, добавляет дату и записывает массив задач в JSON-файл.
6. Сервер возвращает статус `201 Created` и JSON с новой задачей.
7. Клиент снова запрашивает `GET /api/tasks` и перерисовывает список.

В DevTools браузера откройте вкладку Network: там видны URL, метод, статус, заголовки, request payload и response.

## 4. API-контракт

| Метод и путь | Назначение | Успех |
|---|---|---|
| `GET /api/tasks` | получить все задачи | `200 { tasks: [...] }` |
| `POST /api/tasks` | создать задачу; body `{ "title": "..." }` | `201 { task: {...} }` |
| `PATCH /api/tasks/:id` | изменить статус; body `{ "completed": true }` | `200 { task: {...} }` |
| `DELETE /api/tasks/:id` | удалить задачу | `204` |

Ошибки возвращаются как `{ "error": "..." }` со статусом `400` или `404`.

## 5. Запуск

Требуется Node.js 20 или новее.

```powershell
npm test
npm start
```

Откройте <http://localhost:3000>. Для разработки с автоматическим перезапуском:

```powershell
npm run dev
```

Остановить сервер: `Ctrl+C`.

## 6. Git: базовый рабочий цикл

Проверить состояние:

```powershell
git status
git log --oneline
```

После изменения кода:

```powershell
git add .
git commit -m "feat: add task board api and client"
git status
```

Модель Git: рабочая папка → staging area (`git add`) → локальный коммит (`git commit`) → удалённый репозиторий (`git push`).

Полезные команды:

```powershell
git diff                 # изменения до git add
git diff --staged        # изменения после git add
git branch               # ветки
git switch -c feature/name
git switch main
git merge feature/name
```

## 7. Настройка GitHub

1. На <https://github.com/new> создайте пустой репозиторий с именем `client-server-task-board`. Не добавляйте README и `.gitignore`, они уже есть локально.
2. Один раз настройте имя и email автора коммитов, подставив свои данные:

```powershell
git config --global user.name "Ваше имя"
git config --global user.email "email@example.com"
```

3. Выполните локально:

```powershell
git init -b main
git add .
git commit -m "feat: add educational client-server task board"
git remote add origin https://github.com/YOUR_USERNAME/client-server-task-board.git
git push -u origin main
```

При запросе авторизации используйте вход GitHub в браузере или Personal Access Token вместо обычного пароля. Никогда не помещайте токен в код, README или URL remote.

После первого `push` откройте вкладку **Actions** на GitHub: workflow `CI` автоматически проверит проект командой `npm test`.

Проверка связи:

```powershell
git remote -v
git fetch origin
git status
```

## 8. Как довести до «полной реализации»

Следующие итерации лучше делать отдельными ветками и коммитами:

1. `feature/database` — заменить JSON на SQLite/PostgreSQL.
2. `feature/auth` — регистрация, хеширование паролей, сессии или JWT.
3. `feature/validation` — единая схема валидации и корректные ошибки.
4. `feature/frontend` — фильтры, редактирование, loading/error states.
5. `feature/quality` — CI в GitHub Actions, линтер, интеграционные тесты.
6. `feature/deploy` — переменные окружения, production-сборка и размещение клиента/сервера.

Перед каждой отправкой: `npm test`, затем `git add`, осмысленный коммит и `git push`.

## 9. Что происходит при запуске

Команда `npm start` запускает `node server.js`. Node создаёт TCP-сервер, начинает слушать порт `3000` и ждёт соединения. Браузер сначала получает `GET /`, сервер отдаёт `public/index.html`; HTML подключает CSS и JavaScript. После загрузки JavaScript делает `GET /api/tasks`. Все последующие действия идут по тому же HTTP-соединению к серверу, а данные переживают перезапуск благодаря `data/tasks.json`.
