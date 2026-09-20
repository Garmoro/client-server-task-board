import { Module } from '@nestjs/common';
import { SummarizerClient } from './summarizer.client';

@Module({ providers: [SummarizerClient], exports: [SummarizerClient] })
export class SummarizerModule {}
