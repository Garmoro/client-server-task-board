import { BadGatewayException, GatewayTimeoutException, Injectable, NotFoundException } from '@nestjs/common';

export interface SourceMessage {
  id: number;
  sender: string;
  content: string;
  created_at: string;
}

export interface SummaryResponse {
  id: number;
  conversation_id: number;
  summary: string;
  problem: string;
  actions_taken: string[];
  status: string;
  sentiment: string;
  priority: string;
  topics: string[];
  model: string;
  prompt_tokens: number;
  completion_tokens: number;
  llm_duration_ms: number;
  created_at: string;
  updated_at: string;
}

@Injectable()
export class SummarizerClient {
  private readonly baseUrl = (process.env.SUMMARIZER_URL ?? 'http://localhost:8091').replace(/\/$/, '');

  async create(conversationId: number, messages: SourceMessage[], regenerate = false): Promise<SummaryResponse> {
    const path = regenerate ? '/api/v1/summaries/' + conversationId + '/regenerate' : '/api/v1/summaries';
    return (await this.request(path, {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ conversation_id: conversationId, messages }),
    })) as SummaryResponse;
  }

  async latest(conversationId: number): Promise<SummaryResponse> {
    return (await this.request('/api/v1/summaries/' + conversationId)) as SummaryResponse;
  }

  async history(conversationId: number): Promise<{ items: SummaryResponse[] }> {
    return (await this.request('/api/v1/summaries/' + conversationId + '/history')) as { items: SummaryResponse[] };
  }

  async health(): Promise<unknown> { return this.request('/health'); }

  private async request(path: string, init?: RequestInit): Promise<unknown> {
    try {
      const response = await fetch(this.baseUrl + path, init);
      const body = (await response.json()) as Record<string, unknown>;
      if (!response.ok) {
        if (response.status === 404) throw new NotFoundException(body);
        if (response.status === 504) throw new GatewayTimeoutException(body);
        throw new BadGatewayException(body);
      }
      return body;
    } catch (error) {
      if (error instanceof NotFoundException || error instanceof GatewayTimeoutException || error instanceof BadGatewayException) throw error;
      throw new BadGatewayException('Summarizer service is unavailable');
    }
  }
}
