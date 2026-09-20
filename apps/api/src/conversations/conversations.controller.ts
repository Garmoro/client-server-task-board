import { Body, Controller, Get, Param, ParseIntPipe, Post } from '@nestjs/common';
import { CreateConversationDto } from './dto/create-conversation.dto';
import { CreateMessageDto } from './dto/create-message.dto';
import { ConversationsService } from './conversations.service';

@Controller('api/v1/conversations')
export class ConversationsController {
  constructor(private readonly conversations: ConversationsService) {}

  @Get()
  list() {
    return this.conversations.list();
  }

  @Post()
  create(@Body() dto: CreateConversationDto) {
    return this.conversations.create(dto);
  }

  @Get(':id')
  get(@Param('id', ParseIntPipe) id: number) {
    return this.conversations.get(id);
  }

  @Post(':id/messages')
  addMessage(@Param('id', ParseIntPipe) id: number, @Body() dto: CreateMessageDto) {
    return this.conversations.addMessage(id, dto);
  }

  @Post(':id/summarize')
  summarize(@Param('id', ParseIntPipe) id: number) {
    return this.conversations.summarize(id);
  }

  @Get(':id/summary')
  latestSummary(@Param('id', ParseIntPipe) id: number) {
    return this.conversations.latestSummary(id);
  }

  @Get(':id/summary/history')
  summaryHistory(@Param('id', ParseIntPipe) id: number) {
    return this.conversations.summaryHistory(id);
  }

  @Post(':id/summary/regenerate')
  regenerate(@Param('id', ParseIntPipe) id: number) {
    return this.conversations.summarize(id, true);
  }
}
