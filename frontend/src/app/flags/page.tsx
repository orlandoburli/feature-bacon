'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import { flagsAPI } from '@/lib/api';
import { Table, TableHead, TableHeader, TableBody, TableRow, TableCell } from '@/components/Table';
import { StatusBadge, Badge } from '@/components/Badge';
import { Pagination } from '@/components/Pagination';
import { EmptyState } from '@/components/EmptyState';
import { Button } from '@/components/Button';
import { Modal } from '@/components/Modal';
import { FormField, Input, Select, Textarea } from '@/components/FormField';
import { Plus, Trash2, Loader2 } from 'lucide-react';

export default function FlagsPage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [createOpen, setCreateOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);

  const { data, isLoading, error } = useQuery({
    queryKey: ['flags', page],
    queryFn: () => flagsAPI.list(page),
  });

  const deleteMutation = useMutation({
    mutationFn: flagsAPI.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['flags'] });
      setDeleteTarget(null);
    },
  });

  const flags = data?.flags ?? [];
  const pagination = data?.pagination;

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-zinc-900 dark:text-zinc-100">
            Feature Flags
          </h1>
          <p className="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
            Manage feature toggles and rollout rules
          </p>
        </div>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="h-4 w-4" />
          Create Flag
        </Button>
      </div>

      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-400">
          {(error as Error).message}
        </div>
      )}

      {isLoading ? (
        <div className="flex items-center justify-center py-20">
          <Loader2 className="h-6 w-6 animate-spin text-zinc-400" />
        </div>
      ) : flags.length === 0 ? (
        <EmptyState
          title="No feature flags"
          description="Create your first feature flag to get started."
          action={
            <Button onClick={() => setCreateOpen(true)}>
              <Plus className="h-4 w-4" />
              Create Flag
            </Button>
          }
        />
      ) : (
        <>
          <Table>
            <TableHead>
              <TableHeader>Key</TableHeader>
              <TableHeader>Type</TableHeader>
              <TableHeader>Semantics</TableHeader>
              <TableHeader>Status</TableHeader>
              <TableHeader>Description</TableHeader>
              <TableHeader className="text-right">Actions</TableHeader>
            </TableHead>
            <TableBody>
              {flags.map((flag) => (
                <TableRow key={flag.key} onClick={() => router.push(`/flags/${flag.key}`)}>
                  <TableCell>
                    <span className="font-mono text-xs font-semibold text-zinc-900 dark:text-zinc-100">
                      {flag.key}
                    </span>
                  </TableCell>
                  <TableCell>
                    <Badge variant="blue">{flag.type}</Badge>
                  </TableCell>
                  <TableCell>
                    <Badge>{flag.semantics}</Badge>
                  </TableCell>
                  <TableCell>
                    <StatusBadge enabled={flag.enabled} />
                  </TableCell>
                  <TableCell>
                    <span className="max-w-[200px] truncate block">
                      {flag.description || '—'}
                    </span>
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={(e) => {
                        e.stopPropagation();
                        setDeleteTarget(flag.key);
                      }}
                    >
                      <Trash2 className="h-3.5 w-3.5 text-red-500" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          {pagination && (
            <Pagination page={page} totalPages={pagination.totalPages} onPageChange={setPage} />
          )}
        </>
      )}

      <CreateFlagModal open={createOpen} onClose={() => setCreateOpen(false)} />

      <Modal open={!!deleteTarget} onClose={() => setDeleteTarget(null)} title="Delete Flag">
        <p className="text-sm text-zinc-600 dark:text-zinc-400">
          Are you sure you want to delete{' '}
          <span className="font-mono font-semibold text-zinc-900 dark:text-zinc-100">
            {deleteTarget}
          </span>
          ? This action cannot be undone.
        </p>
        <div className="mt-6 flex justify-end gap-3">
          <Button variant="secondary" onClick={() => setDeleteTarget(null)}>
            Cancel
          </Button>
          <Button
            variant="danger"
            loading={deleteMutation.isPending}
            onClick={() => deleteTarget && deleteMutation.mutate(deleteTarget)}
          >
            Delete
          </Button>
        </div>
      </Modal>
    </div>
  );
}

function CreateFlagModal({ open, onClose }: { open: boolean; onClose: () => void }) {
  const queryClient = useQueryClient();
  const [form, setForm] = useState({
    key: '',
    type: 'boolean',
    semantics: 'flag',
    description: '',
    enabled: false,
  });

  const mutation = useMutation({
    mutationFn: (data: typeof form) => flagsAPI.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['flags'] });
      setForm({ key: '', type: 'boolean', semantics: 'flag', description: '', enabled: false });
      onClose();
    },
  });

  return (
    <Modal open={open} onClose={onClose} title="Create Feature Flag">
      <form
        onSubmit={(e) => {
          e.preventDefault();
          mutation.mutate(form);
        }}
        className="space-y-4"
      >
        <FormField label="Key" htmlFor="flag-key">
          <Input
            id="flag-key"
            placeholder="my-feature-flag"
            value={form.key}
            onChange={(e) => setForm({ ...form, key: e.target.value })}
            required
          />
        </FormField>
        <div className="grid grid-cols-2 gap-4">
          <FormField label="Type" htmlFor="flag-type">
            <Select
              id="flag-type"
              value={form.type}
              onChange={(e) => setForm({ ...form, type: e.target.value })}
            >
              <option value="boolean">Boolean</option>
              <option value="string">String</option>
              <option value="number">Number</option>
              <option value="json">JSON</option>
            </Select>
          </FormField>
          <FormField label="Semantics" htmlFor="flag-semantics">
            <Select
              id="flag-semantics"
              value={form.semantics}
              onChange={(e) => setForm({ ...form, semantics: e.target.value })}
            >
              <option value="flag">Flag</option>
              <option value="experiment">Experiment</option>
            </Select>
          </FormField>
        </div>
        <FormField label="Description" htmlFor="flag-description">
          <Textarea
            id="flag-description"
            rows={2}
            placeholder="What does this flag control?"
            value={form.description}
            onChange={(e) => setForm({ ...form, description: e.target.value })}
          />
        </FormField>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={form.enabled}
            onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
            className="rounded border-zinc-300 text-zinc-900 focus:ring-zinc-500 dark:border-zinc-600"
          />
          <span className="text-zinc-700 dark:text-zinc-300">Enable immediately</span>
        </label>

        {mutation.error && (
          <p className="text-sm text-red-500">{(mutation.error as Error).message}</p>
        )}

        <div className="flex justify-end gap-3 pt-2">
          <Button variant="secondary" type="button" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" loading={mutation.isPending}>
            Create Flag
          </Button>
        </div>
      </form>
    </Modal>
  );
}
