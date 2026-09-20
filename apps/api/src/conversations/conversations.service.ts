import { BadRequestException, Injectable, NotFoundException } from '@nestjs/common';
import { PrismaService } from '../prisma/prisma.service';
import { SourceMessage, SummarizerClient } from '../summarizer/summarizer.client';
import { CreateConversationDto } from './dto/create-conversation.dto';
import { CreateMessageDto } from './dto/create-message.dto';

@Injectable()
export class ConversationsService {
  constructor(
    private readonly prisma: PrismaService,
    private readonly summarizer: SummarizerClient,
  ) {}

  async list() {
    return this.prisma.conversation.findMany({
      orderBy: { updatedAt: 'desc' },
      include: { _count: { select: { messages: true } } },
    });
  }

  async create(dto: CreateConversationDto) {
    return this.prisma.conversation.create({
      data: { title: dto.title?.trim() || null },
      include: { messages: true },
    });
  }

  async get(conversationId: number) {
    const conversation = await this.prisma.conversation.findUnique({
      where: { id: conversationId },
      include: { messages: { orderBy: { createdAt: 'asc' } } },
    });
    if (!conversation) throw new NotFoundException('Conversation not found');
    return conversation;
  }

  async addMessage(conversationId: number, dto: CreateMessageDto) {
    await this.get(conversationId);
    const content = dto.content.trim();
    if (!content) throw new BadRequestException('Message content must not be empty');
    const message = await this.prisma.message.create({
      data: { conversationId, sender: dto.sender, content },
    });
    return message;
  }

  async summarize(conversationId: number, regenerate = false) {
    const conversation = await this.get(conversationId);
    if (conversation.messages.length === 0) {
      throw new BadRequestException('Add at least one message before summarizing');
    }
    const messages: SourceMessage[] = conversation.messages.map((message) => ({
      id: message.id,
      sender: message.sender,
      content: message.content,
      created_at: message.createdAt.toISOString(),
    }));
    return this.summarizer.create(conversationId, messages, regenerate);
  }

  async latestSummary(conversationId: number) {
    await this.get(conversationId);
    return this.summarizer.latest(conversationId);
  }

  async summaryHistory(conversationId: number) {
    await this.get(conversationId);
    return this.summarizer.history(conversationId);
  }
}
