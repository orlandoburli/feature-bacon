'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import { experimentsAPI } from '@/lib/api';
import { Table, TableHead, TableHeader, TableBody, TableRow, TableCell } from '@/components/Table';
import { ExperimentStatusBadge, Badge } from '@/components/Badge';
import { Pagination } from '@/components/Pagination';
import { EmptyState } from '@/components/EmptyState';
import { Button } from '@/components/Button';
import { Modal } from '@/components/Modal';
import { FormField, Input } from '@/components/FormField';
import { Plus, Play, Pause, CheckCircle2, Loader2 } from 'lucide-react';

export default function ExperimentsPage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [createOpen, setCreateOpen] = useState(false);

  const { data, isLoading, error } = useQuery({
    queryKey: ['experiments', page],
    queryFn: () => experimentsAPI.list(page),
  });

  const startMutation = useMutation({
    mutationFn: experimentsAPI.start,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['experiments'] }),
  });

  const pauseMutation = useMutation({
    mutationFn: experimentsAPI.pause,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['experiments'] }),
  });

  const completeMutation = useMutation({
    mutationFn: experimentsAPI.complete,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['experiments'] }),
  });

  const experiments = data?.experiments ?? [];
  const pagination = data?.pagination;

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-zinc-900 dark:text-zinc-100">
            Experiments
          </h1>
          <p className="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
            Run A/B tests and manage experiment lifecycles
          </p>
        </div>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="h-4 w-4" />
          Create Experiment
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
      ) : experiments.length === 0 ? (
        <EmptyState
          title="No experiments"
          description="Create your first experiment to start A/B testing."
          action={
            <Button onClick={() => setCreateOpen(true)}>
              <Plus className="h-4 w-4" />
              Create Experiment
            </Button>
          }
        />
      ) : (
        <>
          <Table>
            <TableHead>
              <TableHeader>Key</TableHeader>
              <TableHeader>Name</TableHeader>
              <TableHeader>Status</TableHeader>
              <TableHeader>Sticky</TableHeader>
              <TableHeader className="text-right">Actions</TableHeader>
            </TableHead>
            <TableBody>
              {experiments.map((exp) => (
                <TableRow key={exp.key} onClick={() => router.push(`/experiments/${exp.key}`)}>
                  <TableCell>
                    <span className="font-mono text-xs font-semibold text-zinc-900 dark:text-zinc-100">
                      {exp.key}
                    </span>
                  </TableCell>
                  <TableCell>{exp.name}</TableCell>
                  <TableCell>
                    <ExperimentStatusBadge status={exp.status} />
                  </TableCell>
                  <TableCell>
                    <Badge variant={exp.stickyAssignment ? 'green' : 'gray'}>
                      {exp.stickyAssignment ? 'Yes' : 'No'}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex items-center justify-end gap-1" onClick={(e) => e.stopPropagation()}>
                      {(exp.status === 'draft' || exp.status === 'paused') && (
                        <Button
                          variant="ghost"
                          size="sm"
                          loading={startMutation.isPending}
                          onClick={() => startMutation.mutate(exp.key)}
                        >
                          <Play className="h-3.5 w-3.5 text-emerald-500" />
                        </Button>
                      )}
                      {exp.status === 'running' && (
                        <Button
                          variant="ghost"
                          size="sm"
                          loading={pauseMutation.isPending}
                          onClick={() => pauseMutation.mutate(exp.key)}
                        >
                          <Pause className="h-3.5 w-3.5 text-amber-500" />
                        </Button>
                      )}
                      {(exp.status === 'running' || exp.status === 'paused') && (
                        <Button
                          variant="ghost"
                          size="sm"
                          loading={completeMutation.isPending}
                          onClick={() => completeMutation.mutate(exp.key)}
                        >
                          <CheckCircle2 className="h-3.5 w-3.5 text-blue-500" />
                        </Button>
                      )}
                    </div>
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

      <CreateExperimentModal open={createOpen} onClose={() => setCreateOpen(false)} />
    </div>
  );
}

function CreateExperimentModal({ open, onClose }: { open: boolean; onClose: () => void }) {
  const queryClient = useQueryClient();
  const [form, setForm] = useState({
    key: '',
    name: '',
    stickyAssignment: true,
    variants: [
      { key: 'control', description: 'Control group' },
      { key: 'treatment', description: 'Treatment group' },
    ],
    allocation: [
      { variantKey: 'control', percentage: 50 },
      { variantKey: 'treatment', percentage: 50 },
    ],
  });

  const mutation = useMutation({
    mutationFn: (data: typeof form) => experimentsAPI.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['experiments'] });
      setForm({
        key: '',
        name: '',
        stickyAssignment: true,
        variants: [
          { key: 'control', description: 'Control group' },
          { key: 'treatment', description: 'Treatment group' },
        ],
        allocation: [
          { variantKey: 'control', percentage: 50 },
          { variantKey: 'treatment', percentage: 50 },
        ],
      });
      onClose();
    },
  });

  return (
    <Modal open={open} onClose={onClose} title="Create Experiment" wide>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          mutation.mutate(form);
        }}
        className="space-y-4"
      >
        <div className="grid grid-cols-2 gap-4">
          <FormField label="Key" htmlFor="exp-key">
            <Input
              id="exp-key"
              placeholder="checkout-redesign"
              value={form.key}
              onChange={(e) => setForm({ ...form, key: e.target.value })}
              required
            />
          </FormField>
          <FormField label="Name" htmlFor="exp-name">
            <Input
              id="exp-name"
              placeholder="Checkout Redesign"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              required
            />
          </FormField>
        </div>

        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={form.stickyAssignment}
            onChange={(e) => setForm({ ...form, stickyAssignment: e.target.checked })}
            className="rounded border-zinc-300 text-zinc-900 focus:ring-zinc-500 dark:border-zinc-600"
          />
          <span className="text-zinc-700 dark:text-zinc-300">Sticky assignment</span>
        </label>

        {mutation.error && (
          <p className="text-sm text-red-500">{(mutation.error as Error).message}</p>
        )}

        <div className="flex justify-end gap-3 pt-2">
          <Button variant="secondary" type="button" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" loading={mutation.isPending}>
            Create Experiment
          </Button>
        </div>
      </form>
    </Modal>
  );
}
