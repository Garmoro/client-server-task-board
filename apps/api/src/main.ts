import { ValidationPipe } from '@nestjs/common';
import { NestFactory } from '@nestjs/core';
import { DocumentBuilder, SwaggerModule } from '@nestjs/swagger';
import { AppModule } from './app.module';

async function bootstrap(): Promise<void> {
  const app = await NestFactory.create(AppModule);

  // Node.js 18+ по умолчанию рубит запросы, которые не завершились за 5 минут.
  // Снимаем лимит, чтобы дождаться ответа от LLM на CPU.
  const server = app.getHttpServer();
  server.requestTimeout = 0;   // 0 = отключено
  server.headersTimeout = 0;

  app.enableCors({ origin: process.env.WEB_ORIGIN ?? 'http://localhost:3000' });
  app.useGlobalPipes(new ValidationPipe({ whitelist: true, transform: true }));
  const swaggerConfig = new DocumentBuilder()
    .setTitle('Conversation Workspace API')
    .setDescription('NestJS BFF for conversations and Go-powered summaries')
    .setVersion('1.0')
    .build();
  SwaggerModule.setup('docs', app, SwaggerModule.createDocument(app, swaggerConfig));
  await app.listen(Number(process.env.PORT ?? 4000), '0.0.0.0');
}

void bootstrap();