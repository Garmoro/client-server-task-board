import { Module } from '@nestjs/common';
import { ConversationsModule } from './conversations/conversations.module';
import { HealthModule } from './health/health.module';
import { PrismaModule } from './prisma/prisma.module';
import { SummarizerModule } from './summarizer/summarizer.module';

@Module({
  imports: [PrismaModule, SummarizerModule, ConversationsModule, HealthModule],
})
export class AppModule {}
