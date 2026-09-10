const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs/promises');
const { createServer, DATA_FILE } = require('../server');

let server;
let baseUrl;

test.before(async () => {
  await fs.rm(DATA_FILE, { force: true });
  server = createServer();
  await new Promise((resolve) => server.listen(0, resolve));
  baseUrl = `http://localhost:${server.address().port}`;
});

test.after(async () => {
  await new Promise((resolve, reject) => server.close((error) => error ? reject(error) : resolve()));
  await fs.rm(DATA_FILE, { force: true });
});

test('GET /api/tasks returns the initial task', async () => {
  const response = await fetch(`${baseUrl}/api/tasks`);
  const body = await response.json();
  assert.equal(response.status, 200);
  assert.equal(body.tasks.length, 1);
  assert.equal(body.tasks[0].id, 'welcome');
});

test('POST, PATCH and DELETE form a complete task lifecycle', async () => {
  const createdResponse = await fetch(`${baseUrl}/api/tasks`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title: 'Проверить полный цикл' })
  });
  const created = await createdResponse.json();
  assert.equal(createdResponse.status, 201);

  const patchedResponse = await fetch(`${baseUrl}/api/tasks/${created.task.id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ completed: true })
  });
  assert.equal(patchedResponse.status, 200);
  assert.equal((await patchedResponse.json()).task.completed, true);

  const deletedResponse = await fetch(`${baseUrl}/api/tasks/${created.task.id}`, { method: 'DELETE' });
  assert.equal(deletedResponse.status, 204);
});

test('POST validates the title', async () => {
  const response = await fetch(`${baseUrl}/api/tasks`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title: '   ' })
  });
  assert.equal(response.status, 400);
});
