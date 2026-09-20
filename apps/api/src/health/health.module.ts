import { Controller, Get, Module, Res } from '@nestjs/common';
import type { Response } from 'express';
import { PrismaService } from '../prisma/prisma.service';
import { SummarizerClient } from '../summarizer/summarizer.client';
import { SummarizerModule } from '../summarizer/summarizer.module';

@Controller('health')
class HealthController {
  constructor(
    private readonly prisma: PrismaService,
    private readonly summarizer: SummarizerClient,
  ) {}

  @Get()
  async check(@Res({ passthrough: true }) response: Response) {
    let database: 'up' | 'down' = 'up';
    let summarizer: 'up' | 'down' = 'up';
    try {
      await this.prisma.$queryRawUnsafe('SELECT 1');
    } catch {
      database = 'down';
    }
    try {
      await this.summarizer.health();
    } catch {
      summarizer = 'down';
    }
    const status = database === 'up' && summarizer === 'up' ? 'ok' : 'degraded';
    response.status(status === 'ok' ? 200 : 503);
    return { status, database, summarizer };
  }
}

@Module({ imports: [SummarizerModule], controllers: [HealthController] })
export class HealthModule {}
