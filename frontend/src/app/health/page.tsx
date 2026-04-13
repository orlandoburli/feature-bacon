'use client';

import { useQuery } from '@tanstack/react-query';
import { healthAPI } from '@/lib/api';
import { Badge } from '@/components/Badge';
import { CheckCircle2, AlertTriangle, XCircle, Loader2, RefreshCw } from 'lucide-react';
import { Button } from '@/components/Button';
import { clsx } from 'clsx';

const statusConfig = {
  ok: { icon: CheckCircle2, color: 'text-emerald-500', bg: 'bg-emerald-50 dark:bg-emerald-500/10', label: 'Healthy', variant: 'green' as const },
  degraded: { icon: AlertTriangle, color: 'text-amber-500', bg: 'bg-amber-50 dark:bg-amber-500/10', label: 'Degraded', variant: 'yellow' as const },
  error: { icon: XCircle, color: 'text-red-500', bg: 'bg-red-50 dark:bg-red-500/10', label: 'Unhealthy', variant: 'red' as const },
};

export default function HealthPage() {
  const { data, isLoading, error, refetch, dataUpdatedAt } = useQuery({
    queryKey: ['health'],
    queryFn: () => healthAPI.readyz(),
    refetchInterval: 10_000,
  });

  const overall = data?.status ?? 'unknown';
  const config = statusConfig[overall as keyof typeof statusConfig] ?? statusConfig.error;
  const StatusIcon = config.icon;
  const modules = data?.modules ?? {};

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-zinc-900 dark:text-zinc-100">
            System Health
          </h1>
          <p className="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
            Real-time health monitoring — auto-refreshes every 10 seconds
          </p>
        </div>
        <Button variant="secondary" onClick={() => refetch()}>
          <RefreshCw className="h-4 w-4" />
          Refresh
        </Button>
      </div>

      {error && (
        <div className="mb-6 rounded-xl border border-red-200 bg-red-50 p-6 text-center dark:border-red-800 dark:bg-red-900/20">
          <XCircle className="mx-auto mb-2 h-10 w-10 text-red-400" />
          <p className="text-sm font-medium text-red-700 dark:text-red-400">
            Unable to reach health endpoint
          </p>
          <p className="mt-1 text-xs text-red-500">{(error as Error).message}</p>
        </div>
      )}

      {isLoading ? (
        <div className="flex items-center justify-center py-20">
          <Loader2 className="h-6 w-6 animate-spin text-zinc-400" />
        </div>
      ) : data ? (
        <>
          <div className={clsx('mb-8 flex items-center gap-4 rounded-xl border p-6', config.bg,
            overall === 'ok' ? 'border-emerald-200 dark:border-emerald-800' :
            overall === 'degraded' ? 'border-amber-200 dark:border-amber-800' :
            'border-red-200 dark:border-red-800'
          )}>
            <StatusIcon className={clsx('h-10 w-10', config.color)} />
            <div>
              <h2 className="text-lg font-bold text-zinc-900 dark:text-zinc-100">
                System is {config.label}
              </h2>
              <p className="text-sm text-zinc-500 dark:text-zinc-400">
                Last checked: {dataUpdatedAt ? new Date(dataUpdatedAt).toLocaleTimeString() : '—'}
              </p>
            </div>
          </div>

          <h2 className="mb-4 text-sm font-semibold text-zinc-900 dark:text-zinc-100">
            Module Status
          </h2>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {Object.entries(modules).map(([name, mod]) => {
              const modConfig = statusConfig[mod.status as keyof typeof statusConfig] ?? statusConfig.error;
              const ModIcon = modConfig.icon;
              return (
                <div
                  key={name}
                  className="rounded-xl border border-zinc-200 bg-white p-5 transition-shadow hover:shadow-sm dark:border-zinc-800 dark:bg-zinc-900"
                >
                  <div className="mb-3 flex items-center justify-between">
                    <div className="flex items-center gap-2.5">
                      <ModIcon className={clsx('h-5 w-5', modConfig.color)} />
                      <h3 className="text-sm font-semibold capitalize text-zinc-900 dark:text-zinc-100">
                        {name}
                      </h3>
                    </div>
                    <Badge variant={modConfig.variant} dot>
                      {mod.status}
                    </Badge>
                  </div>
                  <dl className="space-y-1.5 text-sm">
                    {mod.latency_ms !== undefined && (
                      <div className="flex justify-between">
                        <dt className="text-zinc-500 dark:text-zinc-400">Latency</dt>
                        <dd className="font-mono text-zinc-900 dark:text-zinc-100">
                          {mod.latency_ms}ms
                        </dd>
                      </div>
                    )}
                    {mod.message && (
                      <div className="flex justify-between">
                        <dt className="text-zinc-500 dark:text-zinc-400">Message</dt>
                        <dd className="max-w-[160px] truncate text-right text-zinc-700 dark:text-zinc-300">
                          {mod.message}
                        </dd>
                      </div>
                    )}
                  </dl>
                </div>
              );
            })}
          </div>
        </>
      ) : null}
    </div>
  );
}
