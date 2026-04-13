'use client';

import { ChevronLeft, ChevronRight } from 'lucide-react';
import { clsx } from 'clsx';

interface PaginationProps {
  page: number;
  totalPages: number;
  onPageChange: (page: number) => void;
}

export function Pagination({ page, totalPages, onPageChange }: PaginationProps) {
  if (totalPages <= 1) return null;

  return (
    <div className="flex items-center justify-between border-t border-zinc-200 px-1 py-3 dark:border-zinc-800">
      <p className="text-sm text-zinc-500 dark:text-zinc-400">
        Page {page} of {totalPages}
      </p>
      <div className="flex gap-1">
        <button
          onClick={() => onPageChange(page - 1)}
          disabled={page <= 1}
          className={clsx(
            'inline-flex items-center rounded-lg px-3 py-1.5 text-sm font-medium transition-colors',
            page <= 1
              ? 'cursor-not-allowed text-zinc-300 dark:text-zinc-600'
              : 'text-zinc-700 hover:bg-zinc-100 dark:text-zinc-300 dark:hover:bg-zinc-800'
          )}
        >
          <ChevronLeft className="mr-1 h-4 w-4" />
          Previous
        </button>
        <button
          onClick={() => onPageChange(page + 1)}
          disabled={page >= totalPages}
          className={clsx(
            'inline-flex items-center rounded-lg px-3 py-1.5 text-sm font-medium transition-colors',
            page >= totalPages
              ? 'cursor-not-allowed text-zinc-300 dark:text-zinc-600'
              : 'text-zinc-700 hover:bg-zinc-100 dark:text-zinc-300 dark:hover:bg-zinc-800'
          )}
        >
          Next
          <ChevronRight className="ml-1 h-4 w-4" />
        </button>
      </div>
    </div>
  );
}
