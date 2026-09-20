import { Module } from '@nestjs/common';
import { SummarizerModule } from '../summarizer/summarizer.module';
import { ConversationsController } from './conversations.controller';
import { ConversationsService } from './conversations.service';

@Module({
  imports: [SummarizerModule],
  controllers: [ConversationsController],
  providers: [ConversationsService],
})
export class ConversationsModule {}
