const form = document.querySelector('#task-form');
const input = document.querySelector('#task-title');
const list = document.querySelector('#task-list');
const counter = document.querySelector('#counter');
const notice = document.querySelector('#notice');

function showNotice(message = '') {
  notice.textContent = message;
}

function render(tasks) {
  const openCount = tasks.filter((task) => !task.completed).length;
  counter.textContent = `${openCount} ${openCount === 1 ? 'активная задача' : 'активных задач'}`;
  list.innerHTML = '';

  if (tasks.length === 0) {
    list.innerHTML = '<li class="empty">Пока пусто. Добавьте первую задачу.</li>';
    return;
  }

  for (const task of tasks) {
    const item = document.createElement('li');
    item.className = `task${task.completed ? ' done' : ''}`;
    item.innerHTML = `
      <label>
        <input type="checkbox" ${task.completed ? 'checked' : ''} aria-label="Отметить задачу выполненной" />
        <span class="task-title"></span>
      </label>
      <button class="delete" type="button" aria-label="Удалить задачу">×</button>`;
    item.querySelector('.task-title').textContent = task.title;
    item.querySelector('input').addEventListener('change', () => updateTask(task.id, item.querySelector('input').checked));
    item.querySelector('.delete').addEventListener('click', () => deleteTask(task.id));
    list.append(item);
  }
}

async function request(url, options) {
  const response = await fetch(url, options);
  const body = response.status === 204 ? null : await response.json();
  if (!response.ok) throw new Error(body?.error || 'Запрос не выполнен');
  return body;
}

async function loadTasks() {
  try {
    const { tasks } = await request('/api/tasks');
    render(tasks);
  } catch (error) {
    showNotice(error.message);
  }
}

form.addEventListener('submit', async (event) => {
  event.preventDefault();
  showNotice('');
  try {
    await request('/api/tasks', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ title: input.value })
    });
    input.value = '';
    await loadTasks();
  } catch (error) {
    showNotice(error.message);
  }
});

async function updateTask(id, completed) {
  try {
    await request(`/api/tasks/${id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ completed })
    });
    await loadTasks();
  } catch (error) {
    showNotice(error.message);
  }
}

async function deleteTask(id) {
  try {
    await request(`/api/tasks/${id}`, { method: 'DELETE' });
    await loadTasks();
  } catch (error) {
    showNotice(error.message);
  }
}

loadTasks();
