'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiKeysAPI } from '@/lib/api';
import { Table, TableHead, TableHeader, TableBody, TableRow, TableCell } from '@/components/Table';
import { ApiKeyStatusBadge, Badge } from '@/components/Badge';
import { Pagination } from '@/components/Pagination';
import { EmptyState } from '@/components/EmptyState';
import { Button } from '@/components/Button';
import { Modal } from '@/components/Modal';
import { FormField, Input, Select } from '@/components/FormField';
import { Plus, Ban, Copy, Check, Loader2 } from 'lucide-react';

export default function ApiKeysPage() {
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [createOpen, setCreateOpen] = useState(false);
  const [revokeTarget, setRevokeTarget] = useState<string | null>(null);
  const [newRawKey, setNewRawKey] = useState<string | null>(null);

  const { data, isLoading, error } = useQuery({
    queryKey: ['apiKeys', page],
    queryFn: () => apiKeysAPI.list(page),
  });

  const revokeMutation = useMutation({
    mutationFn: apiKeysAPI.revoke,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['apiKeys'] });
      setRevokeTarget(null);
    },
  });

  const apiKeys = data?.apiKeys ?? [];
  const pagination = data?.pagination;

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-zinc-900 dark:text-zinc-100">
            API Keys
          </h1>
          <p className="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
            Manage API access credentials for your applications
          </p>
        </div>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="h-4 w-4" />
          Create API Key
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
      ) : apiKeys.length === 0 ? (
        <EmptyState
          title="No API keys"
          description="Create an API key to authenticate your applications."
          action={
            <Button onClick={() => setCreateOpen(true)}>
              <Plus className="h-4 w-4" />
              Create API Key
            </Button>
          }
        />
      ) : (
        <>
          <Table>
            <TableHead>
              <TableHeader>Name</TableHeader>
              <TableHeader>Scope</TableHeader>
              <TableHeader>Key Prefix</TableHeader>
              <TableHeader>Created</TableHeader>
              <TableHeader>Status</TableHeader>
              <TableHeader className="text-right">Actions</TableHeader>
            </TableHead>
            <TableBody>
              {apiKeys.map((key) => (
                <TableRow key={key.id}>
                  <TableCell>
                    <span className="font-medium text-zinc-900 dark:text-zinc-100">{key.name}</span>
                  </TableCell>
                  <TableCell>
                    <Badge variant="purple">{key.scope}</Badge>
                  </TableCell>
                  <TableCell>
                    <span className="font-mono text-xs">{key.keyPrefix}…</span>
                  </TableCell>
                  <TableCell>
                    {key.createdAt
                      ? new Date(key.createdAt * 1000).toLocaleDateString()
                      : '—'}
                  </TableCell>
                  <TableCell>
                    <ApiKeyStatusBadge revoked={!!key.revokedAt} />
                  </TableCell>
                  <TableCell className="text-right">
                    {!key.revokedAt && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setRevokeTarget(key.id)}
                      >
                        <Ban className="h-3.5 w-3.5 text-red-500" />
                      </Button>
                    )}
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

      <CreateApiKeyModal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        onCreated={(rawKey) => {
          setCreateOpen(false);
          setNewRawKey(rawKey);
        }}
      />

      <RawKeyModal rawKey={newRawKey} onClose={() => setNewRawKey(null)} />

      <Modal open={!!revokeTarget} onClose={() => setRevokeTarget(null)} title="Revoke API Key">
        <p className="text-sm text-zinc-600 dark:text-zinc-400">
          Are you sure you want to revoke this API key? Applications using this key will lose access
          immediately.
        </p>
        <div className="mt-6 flex justify-end gap-3">
          <Button variant="secondary" onClick={() => setRevokeTarget(null)}>
            Cancel
          </Button>
          <Button
            variant="danger"
            loading={revokeMutation.isPending}
            onClick={() => revokeTarget && revokeMutation.mutate(revokeTarget)}
          >
            Revoke
          </Button>
        </div>
      </Modal>
    </div>
  );
}

function CreateApiKeyModal({
  open,
  onClose,
  onCreated,
}: {
  open: boolean;
  onClose: () => void;
  onCreated: (rawKey: string) => void;
}) {
  const queryClient = useQueryClient();
  const [form, setForm] = useState({ name: '', scope: 'read:eval' });

  const mutation = useMutation({
    mutationFn: (data: typeof form) => apiKeysAPI.create(data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['apiKeys'] });
      setForm({ name: '', scope: 'read:eval' });
      onCreated(data.rawKey);
    },
  });

  return (
    <Modal open={open} onClose={onClose} title="Create API Key">
      <form
        onSubmit={(e) => {
          e.preventDefault();
          mutation.mutate(form);
        }}
        className="space-y-4"
      >
        <FormField label="Name" htmlFor="key-name">
          <Input
            id="key-name"
            placeholder="My Application"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
            required
          />
        </FormField>
        <FormField label="Scope" htmlFor="key-scope">
          <Select
            id="key-scope"
            value={form.scope}
            onChange={(e) => setForm({ ...form, scope: e.target.value })}
          >
            <option value="read:eval">Read / Evaluate</option>
            <option value="management">Management</option>
          </Select>
        </FormField>

        {mutation.error && (
          <p className="text-sm text-red-500">{(mutation.error as Error).message}</p>
        )}

        <div className="flex justify-end gap-3 pt-2">
          <Button variant="secondary" type="button" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" loading={mutation.isPending}>
            Create Key
          </Button>
        </div>
      </form>
    </Modal>
  );
}

function RawKeyModal({ rawKey, onClose }: { rawKey: string | null; onClose: () => void }) {
  const [copied, setCopied] = useState(false);

  const copyKey = async () => {
    if (!rawKey) return;
    await navigator.clipboard.writeText(rawKey);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <Modal open={!!rawKey} onClose={onClose} title="API Key Created">
      <div className="space-y-4">
        <div className="rounded-lg border border-amber-200 bg-amber-50 p-3 dark:border-amber-700 dark:bg-amber-900/20">
          <p className="text-sm font-medium text-amber-800 dark:text-amber-300">
            Copy this key now — you won&apos;t be able to see it again.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <code className="flex-1 overflow-x-auto rounded-lg bg-zinc-100 px-3 py-2 font-mono text-sm text-zinc-900 dark:bg-zinc-800 dark:text-zinc-100">
            {rawKey}
          </code>
          <Button variant="secondary" size="sm" onClick={copyKey}>
            {copied ? <Check className="h-4 w-4 text-emerald-500" /> : <Copy className="h-4 w-4" />}
          </Button>
        </div>

        <div className="flex justify-end">
          <Button onClick={onClose}>Done</Button>
        </div>
      </div>
    </Modal>
  );
}
