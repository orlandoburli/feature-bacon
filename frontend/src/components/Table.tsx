import { clsx } from 'clsx';

interface TableProps {
  children: React.ReactNode;
  className?: string;
}

export function Table({ children, className }: TableProps) {
  return (
    <div className={clsx('overflow-hidden rounded-xl border border-zinc-200 dark:border-zinc-800', className)}>
      <table className="min-w-full divide-y divide-zinc-200 dark:divide-zinc-800">
        {children}
      </table>
    </div>
  );
}

export function TableHead({ children }: { children: React.ReactNode }) {
  return (
    <thead className="bg-zinc-50 dark:bg-zinc-900/50">
      <tr>{children}</tr>
    </thead>
  );
}

export function TableHeader({ children, className }: TableProps) {
  return (
    <th
      className={clsx(
        'px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-zinc-500 dark:text-zinc-400',
        className
      )}
    >
      {children}
    </th>
  );
}

export function TableBody({ children }: { children: React.ReactNode }) {
  return (
    <tbody className="divide-y divide-zinc-100 bg-white dark:divide-zinc-800/50 dark:bg-zinc-950">
      {children}
    </tbody>
  );
}

export function TableRow({ children, className, onClick }: TableProps & { onClick?: () => void }) {
  return (
    <tr
      className={clsx(
        'transition-colors',
        onClick && 'cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-900/50',
        className
      )}
      onClick={onClick}
    >
      {children}
    </tr>
  );
}

export function TableCell({ children, className }: TableProps) {
  return (
    <td className={clsx('whitespace-nowrap px-4 py-3 text-sm text-zinc-700 dark:text-zinc-300', className)}>
      {children}
    </td>
  );
}
