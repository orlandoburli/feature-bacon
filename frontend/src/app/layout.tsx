import type { Metadata } from 'next';
import { Geist, Geist_Mono } from 'next/font/google';
import './globals.css';
import { Providers } from '@/components/Providers';
import { Sidebar } from '@/components/Sidebar';

const geistSans = Geist({
  variable: '--font-geist-sans',
  subsets: ['latin'],
});

const geistMono = Geist_Mono({
  variable: '--font-geist-mono',
  subsets: ['latin'],
});

export const metadata: Metadata = {
  title: 'Feature Bacon — Management UI',
  description: 'Manage feature flags, experiments, and API keys',
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}>
      <body className="h-full bg-background text-foreground">
        <Providers>
          <Sidebar />
          <main className="min-h-full lg:pl-60">
            <div className="mx-auto max-w-6xl px-6 py-8 lg:px-8">{children}</div>
          </main>
        </Providers>
      </body>
    </html>
  );
}
