import type { Metadata } from 'next';
import './globals.css';

export const metadata: Metadata = {
  title: 'СпросиИИ — Conversation Lab',
  description: 'Клиентская часть сервиса резюмирования диалогов',
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="ru">
      <body>{children}</body>
    </html>
  );
}
