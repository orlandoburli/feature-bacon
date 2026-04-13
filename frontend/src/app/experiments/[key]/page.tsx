'use client';

import { use, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { experimentsAPI } from '@/lib/api';
import type { Experiment } from '@/lib/api';
import { Button } from '@/components/Button';
import { ExperimentStatusBadge, Badge } from '@/components/Badge';
import { Modal } from '@/components/Modal';
import { FormField, Input } from '@/components/FormField';
import { Table, TableHead, TableHeader, TableBody, TableRow, TableCell } from '@/components/Table';
import { ArrowLeft, Pencil, Play, Pause, CheckCircle2, Loader2, Plus, X } from 'lucide-react';
import Link from 'next/link';

export default function ExperimentDetailPage({ params }: { params: Promise<{ key: string }> }) {
  const { key } = use(params);
  const queryClient = useQueryClient();
  const [editOpen, setEditOpen] = useState(false);

  const { data, isLoading, error } = useQuery({
    queryKey: ['experiment', key],
    queryFn: () => experimentsAPI.get(key),
  });

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ['experiment', key] });
    queryClient.invalidateQueries({ queryKey: ['experiments'] });
  };

  const startMutation = useMutation({ mutationFn: () => experimentsAPI.start(key), onSuccess: invalidate });
  const pauseMutation = useMutation({ mutationFn: () => experimentsAPI.pause(key), onSuccess: invalidate });
  const completeMutation = useMutation({ mutationFn: () => experimentsAPI.complete(key), onSuccess: invalidate });

  const experiment = data?.experiment;

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-6 w-6 animate-spin text-zinc-400" />
      </div>
    );
  }

  if (error || !experiment) {
    return (
      <div className="py-10 text-center">
        <p className="text-sm text-red-500">{(error as Error)?.message || 'Experiment not found'}</p>
        <Link href="/experiments" className="mt-2 inline-block text-sm text-zinc-500 hover:text-zinc-700">
          Back to experiments
        </Link>
      </div>
    );
  }

  return (
    <div>
      <Link
        href="/experiments"
        className="mb-6 inline-flex items-center gap-1.5 text-sm text-zinc-500 transition-colors hover:text-zinc-900 dark:hover:text-zinc-100"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to experiments
      </Link>

      <div className="mb-6 flex items-start justify-between">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-xl font-bold text-zinc-900 dark:text-zinc-100">{experiment.name}</h1>
            <ExperimentStatusBadge status={experiment.status} />
          </div>
          <p className="mt-1 font-mono text-sm text-zinc-500 dark:text-zinc-400">{experiment.key}</p>
        </div>
        <div className="flex gap-2">
          {(experiment.status === 'draft' || experiment.status === 'paused') && (
            <Button variant="secondary" loading={startMutation.isPending} onClick={() => startMutation.mutate()}>
              <Play className="h-3.5 w-3.5" />
              Start
            </Button>
          )}
          {experiment.status === 'running' && (
            <Button variant="secondary" loading={pauseMutation.isPending} onClick={() => pauseMutation.mutate()}>
              <Pause className="h-3.5 w-3.5" />
              Pause
            </Button>
          )}
          {(experiment.status === 'running' || experiment.status === 'paused') && (
            <Button variant="secondary" loading={completeMutation.isPending} onClick={() => completeMutation.mutate()}>
              <CheckCircle2 className="h-3.5 w-3.5" />
              Complete
            </Button>
          )}
          <Button variant="secondary" onClick={() => setEditOpen(true)}>
            <Pencil className="h-3.5 w-3.5" />
            Edit
          </Button>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <div className="rounded-xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
          <h2 className="mb-4 text-sm font-semibold text-zinc-900 dark:text-zinc-100">Details</h2>
          <dl className="space-y-3 text-sm">
            <div className="flex justify-between">
              <dt className="text-zinc-500 dark:text-zinc-400">Status</dt>
              <dd><ExperimentStatusBadge status={experiment.status} /></dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-zinc-500 dark:text-zinc-400">Sticky Assignment</dt>
              <dd>
                <Badge variant={experiment.stickyAssignment ? 'green' : 'gray'}>
                  {experiment.stickyAssignment ? 'Yes' : 'No'}
                </Badge>
              </dd>
            </div>
            {experiment.createdAt && (
              <div className="flex justify-between">
                <dt className="text-zinc-500 dark:text-zinc-400">Created</dt>
                <dd className="text-zinc-900 dark:text-zinc-100">
                  {new Date(experiment.createdAt * 1000).toLocaleDateString()}
                </dd>
              </div>
            )}
            {experiment.updatedAt && (
              <div className="flex justify-between">
                <dt className="text-zinc-500 dark:text-zinc-400">Updated</dt>
                <dd className="text-zinc-900 dark:text-zinc-100">
                  {new Date(experiment.updatedAt * 1000).toLocaleDateString()}
                </dd>
              </div>
            )}
          </dl>
        </div>

        <div className="space-y-6">
          <div className="rounded-xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
            <h2 className="mb-4 text-sm font-semibold text-zinc-900 dark:text-zinc-100">Variants</h2>
            {experiment.variants && experiment.variants.length > 0 ? (
              <Table>
                <TableHead>
                  <TableHeader>Key</TableHeader>
                  <TableHeader>Description</TableHeader>
                </TableHead>
                <TableBody>
                  {experiment.variants.map((v) => (
                    <TableRow key={v.key}>
                      <TableCell>
                        <span className="font-mono text-xs font-semibold">{v.key}</span>
                      </TableCell>
                      <TableCell>{v.description || '—'}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            ) : (
              <p className="py-4 text-center text-sm text-zinc-400">No variants configured</p>
            )}
          </div>

          <div className="rounded-xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
            <h2 className="mb-4 text-sm font-semibold text-zinc-900 dark:text-zinc-100">Allocation</h2>
            {experiment.allocation && experiment.allocation.length > 0 ? (
              <>
                <div className="mb-3 flex h-3 overflow-hidden rounded-full">
                  {experiment.allocation.map((a, i) => {
                    const colors = [
                      'bg-blue-500',
                      'bg-emerald-500',
                      'bg-amber-500',
                      'bg-purple-500',
                      'bg-rose-500',
                    ];
                    return (
                      <div
                        key={a.variantKey}
                        className={colors[i % colors.length]}
                        style={{ width: `${a.percentage}%` }}
                        title={`${a.variantKey}: ${a.percentage}%`}
                      />
                    );
                  })}
                </div>
                <div className="space-y-1">
                  {experiment.allocation.map((a, i) => {
                    const colors = [
                      'bg-blue-500',
                      'bg-emerald-500',
                      'bg-amber-500',
                      'bg-purple-500',
                      'bg-rose-500',
                    ];
                    return (
                      <div key={a.variantKey} className="flex items-center gap-2 text-sm">
                        <div className={`h-2.5 w-2.5 rounded-full ${colors[i % colors.length]}`} />
                        <span className="font-mono text-xs text-zinc-700 dark:text-zinc-300">
                          {a.variantKey}
                        </span>
                        <span className="text-zinc-400">—</span>
                        <span className="font-medium text-zinc-900 dark:text-zinc-100">
                          {a.percentage}%
                        </span>
                      </div>
                    );
                  })}
                </div>
              </>
            ) : (
              <p className="py-4 text-center text-sm text-zinc-400">No allocation configured</p>
            )}
          </div>
        </div>
      </div>

      <EditExperimentModal
        open={editOpen}
        onClose={() => setEditOpen(false)}
        experiment={experiment}
        experimentKey={key}
      />
    </div>
  );
}

function EditExperimentModal({
  open,
  onClose,
  experiment,
  experimentKey,
}: {
  open: boolean;
  onClose: () => void;
  experiment: Experiment;
  experimentKey: string;
}) {
  const queryClient = useQueryClient();
  const [form, setForm] = useState({
    name: experiment.name,
    stickyAssignment: experiment.stickyAssignment,
    variants: experiment.variants || [],
    allocation: experiment.allocation || [],
  });

  const mutation = useMutation({
    mutationFn: (data: typeof form) => experimentsAPI.update(experimentKey, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['experiment', experimentKey] });
      queryClient.invalidateQueries({ queryKey: ['experiments'] });
      onClose();
    },
  });

  const addVariant = () => {
    const newKey = `variant-${form.variants.length + 1}`;
    setForm({
      ...form,
      variants: [...form.variants, { key: newKey, description: '' }],
      allocation: [...form.allocation, { variantKey: newKey, percentage: 0 }],
    });
  };

  const removeVariant = (idx: number) => {
    const removed = form.variants[idx];
    setForm({
      ...form,
      variants: form.variants.filter((_, i) => i !== idx),
      allocation: form.allocation.filter((a) => a.variantKey !== removed.key),
    });
  };

  return (
    <Modal open={open} onClose={onClose} title="Edit Experiment" wide>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          mutation.mutate(form);
        }}
        className="space-y-4"
      >
        <FormField label="Name" htmlFor="edit-name">
          <Input
            id="edit-name"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
            required
          />
        </FormField>

        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={form.stickyAssignment}
            onChange={(e) => setForm({ ...form, stickyAssignment: e.target.checked })}
            className="rounded border-zinc-300 text-zinc-900 focus:ring-zinc-500 dark:border-zinc-600"
          />
          <span className="text-zinc-700 dark:text-zinc-300">Sticky assignment</span>
        </label>

        <div>
          <div className="mb-2 flex items-center justify-between">
            <h3 className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">Variants</h3>
            <Button variant="ghost" size="sm" type="button" onClick={addVariant}>
              <Plus className="h-3.5 w-3.5" />
              Add Variant
            </Button>
          </div>
          <div className="space-y-2">
            {form.variants.map((v, i) => (
              <div key={i} className="flex items-center gap-2">
                <Input
                  value={v.key}
                  onChange={(e) => {
                    const oldKey = v.key;
                    const newVariants = [...form.variants];
                    newVariants[i] = { ...v, key: e.target.value };
                    const newAllocation = form.allocation.map((a) =>
                      a.variantKey === oldKey ? { ...a, variantKey: e.target.value } : a
                    );
                    setForm({ ...form, variants: newVariants, allocation: newAllocation });
                  }}
                  placeholder="Variant key"
                  className="flex-1"
                />
                <Input
                  value={v.description || ''}
                  onChange={(e) => {
                    const newVariants = [...form.variants];
                    newVariants[i] = { ...v, description: e.target.value };
                    setForm({ ...form, variants: newVariants });
                  }}
                  placeholder="Description"
                  className="flex-1"
                />
                <Input
                  type="number"
                  min={0}
                  max={100}
                  value={form.allocation.find((a) => a.variantKey === v.key)?.percentage ?? 0}
                  onChange={(e) => {
                    const newAllocation = form.allocation.map((a) =>
                      a.variantKey === v.key ? { ...a, percentage: Number(e.target.value) } : a
                    );
                    setForm({ ...form, allocation: newAllocation });
                  }}
                  className="w-20"
                  placeholder="%"
                />
                <button type="button" onClick={() => removeVariant(i)}>
                  <X className="h-4 w-4 text-zinc-400 hover:text-red-500" />
                </button>
              </div>
            ))}
          </div>
        </div>

        {mutation.error && (
          <p className="text-sm text-red-500">{(mutation.error as Error).message}</p>
        )}

        <div className="flex justify-end gap-3 pt-2">
          <Button variant="secondary" type="button" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" loading={mutation.isPending}>
            Save Changes
          </Button>
        </div>
      </form>
    </Modal>
  );
}
