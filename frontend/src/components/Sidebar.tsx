'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { clsx } from 'clsx';
import { Flag, FlaskConical, Key, Activity, LayoutDashboard, Flame, Menu, X } from 'lucide-react';
import { useState } from 'react';

const navigation = [
  { name: 'Dashboard', href: '/', icon: LayoutDashboard },
  { name: 'Flags', href: '/flags', icon: Flag },
  { name: 'Experiments', href: '/experiments', icon: FlaskConical },
  { name: 'API Keys', href: '/api-keys', icon: Key },
  { name: 'Health', href: '/health', icon: Activity },
];

function NavContent({ pathname, onItemClick }: { pathname: string; onItemClick?: () => void }) {
  return (
    <>
      <div className="flex h-14 items-center gap-2.5 border-b border-zinc-200 px-5 dark:border-zinc-800">
        <Flame className="h-6 w-6 text-orange-500" />
        <span className="text-base font-bold tracking-tight text-zinc-900 dark:text-zinc-100">
          Feature Bacon
        </span>
      </div>
      <nav className="flex-1 space-y-1 px-3 py-4">
        {navigation.map((item) => {
          const isActive = item.href === '/' ? pathname === '/' : pathname.startsWith(item.href);
          return (
            <Link
              key={item.name}
              href={item.href}
              onClick={onItemClick}
              className={clsx(
                'group flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                isActive
                  ? 'bg-zinc-100 text-zinc-900 dark:bg-zinc-800 dark:text-zinc-100'
                  : 'text-zinc-600 hover:bg-zinc-50 hover:text-zinc-900 dark:text-zinc-400 dark:hover:bg-zinc-800/50 dark:hover:text-zinc-100'
              )}
            >
              <item.icon
                className={clsx(
                  'h-4.5 w-4.5 shrink-0',
                  isActive
                    ? 'text-zinc-900 dark:text-zinc-100'
                    : 'text-zinc-400 group-hover:text-zinc-600 dark:text-zinc-500 dark:group-hover:text-zinc-300'
                )}
              />
              {item.name}
            </Link>
          );
        })}
      </nav>
    </>
  );
}

export function Sidebar() {
  const pathname = usePathname();
  const [mobileOpen, setMobileOpen] = useState(false);

  return (
    <>
      <button
        className="fixed left-4 top-4 z-50 rounded-lg bg-white p-2 shadow-md ring-1 ring-zinc-200 lg:hidden dark:bg-zinc-900 dark:ring-zinc-700"
        onClick={() => setMobileOpen(true)}
      >
        <Menu className="h-5 w-5 text-zinc-700 dark:text-zinc-300" />
      </button>

      {mobileOpen && (
        <div className="fixed inset-0 z-40 lg:hidden">
          <div className="fixed inset-0 bg-black/30 backdrop-blur-sm" onClick={() => setMobileOpen(false)} />
          <div className="fixed inset-y-0 left-0 flex w-64 flex-col bg-white dark:bg-zinc-950">
            <button
              className="absolute right-3 top-4 rounded-lg p-1 text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300"
              onClick={() => setMobileOpen(false)}
            >
              <X className="h-5 w-5" />
            </button>
            <NavContent pathname={pathname} onItemClick={() => setMobileOpen(false)} />
          </div>
        </div>
      )}

      <div className="hidden lg:fixed lg:inset-y-0 lg:flex lg:w-60 lg:flex-col">
        <div className="flex grow flex-col border-r border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-950">
          <NavContent pathname={pathname} />
        </div>
      </div>
    </>
  );
}
