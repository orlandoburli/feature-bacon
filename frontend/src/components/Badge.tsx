import { clsx } from 'clsx';

const variants = {
  green: 'bg-emerald-50 text-emerald-700 ring-emerald-600/20 dark:bg-emerald-500/10 dark:text-emerald-400 dark:ring-emerald-500/20',
  red: 'bg-red-50 text-red-700 ring-red-600/20 dark:bg-red-500/10 dark:text-red-400 dark:ring-red-500/20',
  yellow: 'bg-amber-50 text-amber-700 ring-amber-600/20 dark:bg-amber-500/10 dark:text-amber-400 dark:ring-amber-500/20',
  blue: 'bg-blue-50 text-blue-700 ring-blue-600/20 dark:bg-blue-500/10 dark:text-blue-400 dark:ring-blue-500/20',
  gray: 'bg-zinc-50 text-zinc-600 ring-zinc-500/20 dark:bg-zinc-500/10 dark:text-zinc-400 dark:ring-zinc-500/20',
  purple: 'bg-purple-50 text-purple-700 ring-purple-600/20 dark:bg-purple-500/10 dark:text-purple-400 dark:ring-purple-500/20',
} as const;

type BadgeVariant = keyof typeof variants;

interface BadgeProps {
  children: React.ReactNode;
  variant?: BadgeVariant;
  dot?: boolean;
  className?: string;
}

export function Badge({ children, variant = 'gray', dot, className }: BadgeProps) {
  return (
    <span
      className={clsx(
        'inline-flex items-center gap-x-1.5 rounded-md px-2 py-1 text-xs font-medium ring-1 ring-inset',
        variants[variant],
        className
      )}
    >
      {dot && (
        <svg className="h-1.5 w-1.5 fill-current" viewBox="0 0 6 6" aria-hidden="true">
          <circle cx={3} cy={3} r={3} />
        </svg>
      )}
      {children}
    </span>
  );
}

export function StatusBadge({ enabled }: { enabled: boolean }) {
  return (
    <Badge variant={enabled ? 'green' : 'gray'} dot>
      {enabled ? 'Enabled' : 'Disabled'}
    </Badge>
  );
}

const experimentStatusMap: Record<string, BadgeVariant> = {
  draft: 'yellow',
  running: 'blue',
  paused: 'purple',
  completed: 'green',
};

export function ExperimentStatusBadge({ status }: { status: string }) {
  return (
    <Badge variant={experimentStatusMap[status] || 'gray'} dot>
      {status.charAt(0).toUpperCase() + status.slice(1)}
    </Badge>
  );
}

export function ApiKeyStatusBadge({ revoked }: { revoked: boolean }) {
  return (
    <Badge variant={revoked ? 'red' : 'green'} dot>
      {revoked ? 'Revoked' : 'Active'}
    </Badge>
  );
}
