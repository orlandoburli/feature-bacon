'use client';

import { useQuery } from '@tanstack/react-query';
import { flagsAPI, experimentsAPI, apiKeysAPI, healthAPI } from '@/lib/api';
import { Flag, FlaskConical, Key, Activity, ArrowRight, Loader2 } from 'lucide-react';
import Link from 'next/link';
import { clsx } from 'clsx';

const cards = [
  {
    title: 'Feature Flags',
    href: '/flags',
    icon: Flag,
    color: 'text-blue-500',
    bg: 'bg-blue-50 dark:bg-blue-500/10',
    description: 'Manage feature toggles and rollouts',
    queryKey: 'flags',
  },
  {
    title: 'Experiments',
    href: '/experiments',
    icon: FlaskConical,
    color: 'text-purple-500',
    bg: 'bg-purple-50 dark:bg-purple-500/10',
    description: 'Run A/B tests and experiments',
    queryKey: 'experiments',
  },
  {
    title: 'API Keys',
    href: '/api-keys',
    icon: Key,
    color: 'text-amber-500',
    bg: 'bg-amber-50 dark:bg-amber-500/10',
    description: 'Manage API access credentials',
    queryKey: 'apiKeys',
  },
  {
    title: 'Health',
    href: '/health',
    icon: Activity,
    color: 'text-emerald-500',
    bg: 'bg-emerald-50 dark:bg-emerald-500/10',
    description: 'Monitor system health',
    queryKey: 'health',
  },
] as const;

export default function DashboardPage() {
  const flags = useQuery({ queryKey: ['flags'], queryFn: () => flagsAPI.list() });
  const experiments = useQuery({ queryKey: ['experiments'], queryFn: () => experimentsAPI.list() });
  const apiKeys = useQuery({ queryKey: ['apiKeys'], queryFn: () => apiKeysAPI.list() });
  const health = useQuery({ queryKey: ['health'], queryFn: () => healthAPI.readyz() });

  const counts: Record<string, string | number> = {
    flags: flags.data?.pagination?.total ?? '—',
    experiments: experiments.data?.pagination?.total ?? '—',
    apiKeys: apiKeys.data?.apiKeys?.filter((k) => !k.revokedAt).length ?? '—',
    health: health.data?.status === 'ok' ? 'Healthy' : health.data?.status ?? '—',
  };

  const isLoading = flags.isLoading || experiments.isLoading || apiKeys.isLoading || health.isLoading;

  return (
    <div>
      <div className="mb-8">
        <h1 className="text-2xl font-bold tracking-tight text-zinc-900 dark:text-zinc-100">
          Dashboard
        </h1>
        <p className="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
          Overview of your Feature Bacon instance
        </p>
      </div>

      {isLoading && (
        <div className="mb-6 flex items-center gap-2 text-sm text-zinc-500">
          <Loader2 className="h-4 w-4 animate-spin" />
          Loading data…
        </div>
      )}

      <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
        {cards.map((card) => (
          <Link
            key={card.href}
            href={card.href}
            className="group relative overflow-hidden rounded-xl border border-zinc-200 bg-white p-5 transition-all hover:border-zinc-300 hover:shadow-md dark:border-zinc-800 dark:bg-zinc-900 dark:hover:border-zinc-700"
          >
            <div className="flex items-center justify-between">
              <div className={clsx('rounded-lg p-2.5', card.bg)}>
                <card.icon className={clsx('h-5 w-5', card.color)} />
              </div>
              <ArrowRight className="h-4 w-4 text-zinc-300 transition-transform group-hover:translate-x-0.5 group-hover:text-zinc-500 dark:text-zinc-600 dark:group-hover:text-zinc-400" />
            </div>
            <div className="mt-4">
              <p className="text-2xl font-bold text-zinc-900 dark:text-zinc-100">
                {counts[card.queryKey]}
              </p>
              <p className="text-sm font-medium text-zinc-900 dark:text-zinc-100">{card.title}</p>
              <p className="mt-0.5 text-xs text-zinc-500 dark:text-zinc-400">{card.description}</p>
            </div>
          </Link>
        ))}
      </div>
    </div>
  );
}
