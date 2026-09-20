'use client';

import { FormEvent, useEffect, useState } from 'react';

type Conversation = { id: number; title: string | null; _count: { messages: number } };
type Message = { id: number; sender: string; content: string; createdAt: string };
type Summary = { id: number; summary: string; problem: string; status: string; sentiment: string; priority: string; topics: string[]; created_at: string };
type Detail = { id: number; title: string | null; messages: Message[] };

const apiUrl = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:4000';

async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(apiUrl + path, { ...init, headers: { 'content-type': 'application/json', ...(init?.headers ?? {}) } });
  const body = await response.json();
  if (!response.ok) throw new Error(body.message ?? body.error ?? 'Запрос не выполнен');
  return body as T;
}

export default function HomePage() {
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [detail, setDetail] = useState<Detail | null>(null);
  const [summary, setSummary] = useState<Summary | null>(null);
  const [history, setHistory] = useState<Summary[]>([]);
  const [title, setTitle] = useState('');
  const [message, setMessage] = useState('');
  const [sender, setSender] = useState('user');
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState('');

  async function loadConversations() {
    setConversations(await api<Conversation[]>('/api/v1/conversations'));
  }

  async function selectConversation(id: number) {
  setSelectedId(id);
  setNotice('');
  const [nextDetail, nextSummary, nextHistory] = await Promise.all([
    api<Detail>('/api/v1/conversations/' + id),
    api<Summary | null>('/api/v1/conversations/' + id + '/summary').catch(() => null),
    api<{ items: Summary[] } | null>('/api/v1/conversations/' + id + '/summary/history').catch(() => null),
  ]);
  setDetail(nextDetail);
  setSummary(nextSummary ?? null);
  setHistory(nextHistory && Array.isArray(nextHistory.items) ? nextHistory.items : []);
}

  useEffect(() => {
    void loadConversations().catch(() => setNotice('NestJS API пока недоступен'));
  }, []);

  async function createConversation(event: FormEvent) {
    event.preventDefault();
    setBusy(true);
    try {
      const created = await api<Conversation>('/api/v1/conversations', { method: 'POST', body: JSON.stringify({ title }) });
      setTitle('');
      await loadConversations();
      await selectConversation(created.id);
    } catch (error) { setNotice(error instanceof Error ? error.message : 'Не удалось создать диалог'); }
    finally { setBusy(false); }
  }

  async function addMessage(event: FormEvent) {
    event.preventDefault();
    if (!selectedId) return;
    setBusy(true);
    try {
      await api('/api/v1/conversations/' + selectedId + '/messages', { method: 'POST', body: JSON.stringify({ sender, content: message }) });
      setMessage('');
      await selectConversation(selectedId);
      await loadConversations();
    } catch (error) { setNotice(error instanceof Error ? error.message : 'Не удалось добавить сообщение'); }
    finally { setBusy(false); }
  }

  async function summarize(regenerate = false) {
    if (!selectedId) return;
    setBusy(true);
    setNotice('');
    try {
      const nextSummary = await api<Summary>('/api/v1/conversations/' + selectedId + (regenerate ? '/summary/regenerate' : '/summarize'), { method: 'POST' });
      setSummary(nextSummary);
      const nextHistory = await api<{ items: Summary[] }>('/api/v1/conversations/' + selectedId + '/summary/history');
      setHistory(nextHistory.items);
    } catch (error) { setNotice(error instanceof Error ? error.message : 'Не удалось получить резюме'); }
    finally { setBusy(false); }
  }

  return (
    <main className="app-shell">
      <aside className="sidebar">
        <p className="eyebrow">СПРОСИИИ / LAB 02</p>
        <h1>Conversation<br /><em>Summarizer.</em></h1>
        <p className="muted">Next.js-клиент → NestJS API → Prisma → Go + LLM.</p>
        <form className="new-chat" onSubmit={createConversation}>
          <input value={title} onChange={(event) => setTitle(event.target.value)} placeholder="Название диалога" maxLength={160} />
          <button disabled={busy}>Новый диалог</button>
        </form>
        <div className="conversation-list">
          {conversations.map((item) => (
            <button className={selectedId === item.id ? 'conversation active' : 'conversation'} key={item.id} onClick={() => void selectConversation(item.id)}>
              <span>{item.title || 'Без названия'}</span><small>{item._count.messages} сообщ.</small>
            </button>
          ))}
        </div>
      </aside>
      <section className="workspace">
        <header className="topbar"><span>КОНТРОЛЬНАЯ ПАНЕЛЬ</span><span className="status-dot">● API connected</span></header>
        {!detail ? <div className="empty-state"><span>01</span><h2>Выберите диалог,<br />чтобы начать анализ.</h2><p>Создайте разговор слева и добавьте несколько сообщений.</p></div> : (
          <div className="content-grid">
            <section className="conversation-panel">
              <div className="section-heading"><div><p className="kicker">ДИАЛОГ #{detail.id}</p><h2>{detail.title || 'Без названия'}</h2></div><span>{detail.messages.length} сообщений</span></div>
              <div className="messages">
                {detail.messages.map((item) => <article className={'message ' + item.sender} key={item.id}><div className="message-meta">{item.sender} · {new Date(item.createdAt).toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })}</div><p>{item.content}</p></article>)}
              </div>
              <form className="message-form" onSubmit={addMessage}>
                <select value={sender} onChange={(event) => setSender(event.target.value)}><option value="user">user</option><option value="operator">operator</option><option value="bot">bot</option></select>
                <input value={message} onChange={(event) => setMessage(event.target.value)} placeholder="Добавьте реплику..." maxLength={20000} required />
                <button disabled={busy}>Отправить</button>
              </form>
            </section>
            <aside className="summary-panel">
              <div className="section-heading"><div><p className="kicker">LLM OUTPUT</p><h2>Резюме</h2></div><span>{history.length} версий</span></div>
              {summary ? <><div className="summary-copy"><p>{summary.summary}</p><dl><div><dt>ПРОБЛЕМА</dt><dd>{summary.problem}</dd></div><div><dt>СТАТУС</dt><dd>{summary.status}</dd></div></dl></div><div className="badges"><span>{summary.priority}</span><span>{summary.sentiment}</span>{summary.topics.map((topic) => <span key={topic}>{topic}</span>)}</div><button className="regenerate" onClick={() => void summarize(true)} disabled={busy}>↻ Повторить генерацию</button></> : <div className="summary-empty"><p>Резюме ещё не создано.</p><button onClick={() => void summarize()} disabled={busy}>Запустить анализ</button></div>}
              {notice && <p className="notice">{notice}</p>}
              {history.length > 0 && <div className="history"><p className="kicker">ИСТОРИЯ</p>{history.map((item) => <div className="history-item" key={item.id}><span>v{item.id}</span><span>{item.priority}</span><time>{new Date(item.created_at).toLocaleDateString('ru-RU')}</time></div>)}</div>}
            </aside>
          </div>
        )}
      </section>
    </main>
  );
}
