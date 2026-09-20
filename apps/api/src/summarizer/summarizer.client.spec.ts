import { SummarizerClient } from './summarizer.client';

describe('SummarizerClient', () => {
  afterEach(() => jest.restoreAllMocks());

  it('sends the conversation contract to Go summarizer', async () => {
    const response = { id: 1, conversation_id: 7, summary: 'Done' };
    jest.spyOn(global, 'fetch').mockResolvedValue(new Response(JSON.stringify(response), { status: 201 }));
    const client = new SummarizerClient();

    await expect(client.create(7, [{
      id: 1,
      sender: 'user',
      content: 'Hello',
      created_at: '2026-09-16T10:00:00.000Z',
    }])).resolves.toEqual(response);
    expect(fetch).toHaveBeenCalledWith('http://localhost:8091/api/v1/summaries', expect.objectContaining({
      method: 'POST',
      body: expect.stringContaining('"conversation_id":7'),
    }));
  });
});
