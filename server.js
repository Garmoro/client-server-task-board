const http = require('node:http');
const fs = require('node:fs/promises');
const path = require('node:path');
const crypto = require('node:crypto');
const { URL } = require('node:url');

const PORT = Number(process.env.PORT) || 3000;
const ROOT = __dirname;
const PUBLIC_DIR = path.join(ROOT, 'public');
const DATA_DIR = path.join(ROOT, 'data');
const DATA_FILE = path.join(DATA_DIR, 'tasks.json');

const initialTasks = [
  {
    id: 'welcome',
    title: 'Изучить путь запроса от браузера до сервера',
    completed: false,
    createdAt: new Date().toISOString()
  }
];

async function ensureDataFile() {
  await fs.mkdir(DATA_DIR, { recursive: true });
  try {
    await fs.access(DATA_FILE);
  } catch {
    await writeTasks(initialTasks);
  }
}

async function readTasks() {
  await ensureDataFile();
  const content = await fs.readFile(DATA_FILE, 'utf8');
  return JSON.parse(content);
}

async function writeTasks(tasks) {
  await fs.writeFile(DATA_FILE, JSON.stringify(tasks, null, 2) + '\n', 'utf8');
}

function sendJson(response, statusCode, payload) {
  response.writeHead(statusCode, {
    'Content-Type': 'application/json; charset=utf-8',
    'Cache-Control': 'no-store'
  });
  response.end(JSON.stringify(payload));
}

function sendError(response, statusCode, message) {
  sendJson(response, statusCode, { error: message });
}

function readBody(request) {
  return new Promise((resolve, reject) => {
    let body = '';
    request.on('data', (chunk) => {
      body += chunk;
      if (body.length > 10_000) {
        reject(new Error('request body is too large'));
        request.destroy();
      }
    });
    request.on('end', () => resolve(body));
    request.on('error', reject);
  });
}

async function parseJsonBody(request, response) {
  try {
    return JSON.parse(await readBody(request));
  } catch {
    sendError(response, 400, 'Тело запроса должно быть корректным JSON');
    return null;
  }
}

async function handleApi(request, response, url) {
  if (url.pathname === '/api/tasks' && request.method === 'GET') {
    return sendJson(response, 200, { tasks: await readTasks() });
  }

  if (url.pathname === '/api/tasks' && request.method === 'POST') {
    const body = await parseJsonBody(request, response);
    if (!body) return;
    const title = typeof body.title === 'string' ? body.title.trim() : '';
    if (!title || title.length > 120) {
      return sendError(response, 400, 'Название задачи: от 1 до 120 символов');
    }

    const tasks = await readTasks();
    const task = {
      id: crypto.randomUUID(),
      title,
      completed: false,
      createdAt: new Date().toISOString()
    };
    tasks.unshift(task);
    await writeTasks(tasks);
    return sendJson(response, 201, { task });
  }

  const taskMatch = url.pathname.match(/^\/api\/tasks\/([^/]+)$/);
  if (taskMatch && request.method === 'PATCH') {
    const body = await parseJsonBody(request, response);
    if (!body) return;
    if (typeof body.completed !== 'boolean') {
      return sendError(response, 400, 'Поле completed должно быть boolean');
    }

    const tasks = await readTasks();
    const task = tasks.find((item) => item.id === taskMatch[1]);
    if (!task) return sendError(response, 404, 'Задача не найдена');
    task.completed = body.completed;
    await writeTasks(tasks);
    return sendJson(response, 200, { task });
  }

  if (taskMatch && request.method === 'DELETE') {
    const tasks = await readTasks();
    const remaining = tasks.filter((item) => item.id !== taskMatch[1]);
    if (remaining.length === tasks.length) {
      return sendError(response, 404, 'Задача не найдена');
    }
    await writeTasks(remaining);
    response.writeHead(204);
    return response.end();
  }

  return sendError(response, 404, 'API-маршрут не найден');
}

async function serveStatic(response, url) {
  const requestedPath = url.pathname === '/' ? '/index.html' : url.pathname;
  const filePath = path.resolve(PUBLIC_DIR, `.${requestedPath}`);
  if (!filePath.startsWith(`${PUBLIC_DIR}${path.sep}`)) {
    response.writeHead(403);
    return response.end('Forbidden');
  }

  try {
    const file = await fs.readFile(filePath);
    const extension = path.extname(filePath);
    const types = {
      '.html': 'text/html; charset=utf-8',
      '.css': 'text/css; charset=utf-8',
      '.js': 'text/javascript; charset=utf-8'
    };
    response.writeHead(200, { 'Content-Type': types[extension] || 'application/octet-stream' });
    response.end(file);
  } catch {
    sendError(response, 404, 'Файл не найден');
  }
}

function createServer() {
  return http.createServer(async (request, response) => {
    try {
      const url = new URL(request.url, `http://${request.headers.host || 'localhost'}`);
      if (url.pathname.startsWith('/api/')) {
        await handleApi(request, response, url);
      } else if (request.method === 'GET') {
        await serveStatic(response, url);
      } else {
        sendError(response, 405, 'Метод не поддерживается');
      }
    } catch (error) {
      console.error(error);
      sendError(response, 500, 'Внутренняя ошибка сервера');
    }
  });
}

if (require.main === module) {
  ensureDataFile().then(() => {
    createServer().listen(PORT, () => {
      console.log(`Task Board запущен: http://localhost:${PORT}`);
    });
  });
}

module.exports = { createServer, DATA_FILE };
