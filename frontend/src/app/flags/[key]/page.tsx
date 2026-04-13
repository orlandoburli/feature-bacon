'use client';

import { use, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import { flagsAPI } from '@/lib/api';
import type { Flag, Rule, Condition } from '@/lib/api';
import { Button } from '@/components/Button';
import { Badge, StatusBadge } from '@/components/Badge';
import { Modal } from '@/components/Modal';
import { FormField, Input, Textarea } from '@/components/FormField';
import { ArrowLeft, Pencil, Trash2, Loader2, Plus, X } from 'lucide-react';
import Link from 'next/link';

export default function FlagDetailPage({ params }: { params: Promise<{ key: string }> }) {
  const { key } = use(params);
  const router = useRouter();
  const queryClient = useQueryClient();
  const [editOpen, setEditOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);

  const { data, isLoading, error } = useQuery({
    queryKey: ['flag', key],
    queryFn: () => flagsAPI.get(key),
  });

  const deleteMutation = useMutation({
    mutationFn: () => flagsAPI.delete(key),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['flags'] });
      router.push('/flags');
    },
  });

  const flag = data?.flag;

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-6 w-6 animate-spin text-zinc-400" />
      </div>
    );
  }

  if (error || !flag) {
    return (
      <div className="py-10 text-center">
        <p className="text-sm text-red-500">{(error as Error)?.message || 'Flag not found'}</p>
        <Link href="/flags" className="mt-2 inline-block text-sm text-zinc-500 hover:text-zinc-700">
          Back to flags
        </Link>
      </div>
    );
  }

  return (
    <div>
      <Link
        href="/flags"
        className="mb-6 inline-flex items-center gap-1.5 text-sm text-zinc-500 transition-colors hover:text-zinc-900 dark:hover:text-zinc-100"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to flags
      </Link>

      <div className="mb-6 flex items-start justify-between">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="font-mono text-xl font-bold text-zinc-900 dark:text-zinc-100">
              {flag.key}
            </h1>
            <StatusBadge enabled={flag.enabled} />
          </div>
          {flag.description && (
            <p className="mt-1 text-sm text-zinc-500 dark:text-zinc-400">{flag.description}</p>
          )}
        </div>
        <div className="flex gap-2">
          <Button variant="secondary" onClick={() => setEditOpen(true)}>
            <Pencil className="h-3.5 w-3.5" />
            Edit
          </Button>
          <Button variant="danger" onClick={() => setDeleteOpen(true)}>
            <Trash2 className="h-3.5 w-3.5" />
            Delete
          </Button>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <div className="rounded-xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
          <h2 className="mb-4 text-sm font-semibold text-zinc-900 dark:text-zinc-100">Details</h2>
          <dl className="space-y-3 text-sm">
            <div className="flex justify-between">
              <dt className="text-zinc-500 dark:text-zinc-400">Type</dt>
              <dd><Badge variant="blue">{flag.type}</Badge></dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-zinc-500 dark:text-zinc-400">Semantics</dt>
              <dd><Badge>{flag.semantics}</Badge></dd>
            </div>
            {flag.defaultResult && (
              <>
                <div className="flex justify-between">
                  <dt className="text-zinc-500 dark:text-zinc-400">Default Enabled</dt>
                  <dd className="text-zinc-900 dark:text-zinc-100">
                    {flag.defaultResult.enabled ? 'Yes' : 'No'}
                  </dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-zinc-500 dark:text-zinc-400">Default Variant</dt>
                  <dd className="font-mono text-zinc-900 dark:text-zinc-100">
                    {flag.defaultResult.variant || '—'}
                  </dd>
                </div>
              </>
            )}
            {flag.createdAt && (
              <div className="flex justify-between">
                <dt className="text-zinc-500 dark:text-zinc-400">Created</dt>
                <dd className="text-zinc-900 dark:text-zinc-100">
                  {new Date(flag.createdAt * 1000).toLocaleDateString()}
                </dd>
              </div>
            )}
            {flag.updatedAt && (
              <div className="flex justify-between">
                <dt className="text-zinc-500 dark:text-zinc-400">Updated</dt>
                <dd className="text-zinc-900 dark:text-zinc-100">
                  {new Date(flag.updatedAt * 1000).toLocaleDateString()}
                </dd>
              </div>
            )}
          </dl>
        </div>

        <div className="rounded-xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
          <h2 className="mb-4 text-sm font-semibold text-zinc-900 dark:text-zinc-100">Rules</h2>
          {flag.rules && flag.rules.length > 0 ? (
            <div className="space-y-3">
              {flag.rules.map((rule, i) => (
                <RuleCard key={i} rule={rule} index={i} />
              ))}
            </div>
          ) : (
            <p className="py-4 text-center text-sm text-zinc-400">No rules configured</p>
          )}
        </div>
      </div>

      <EditFlagModal open={editOpen} onClose={() => setEditOpen(false)} flag={flag} flagKey={key} />

      <Modal open={deleteOpen} onClose={() => setDeleteOpen(false)} title="Delete Flag">
        <p className="text-sm text-zinc-600 dark:text-zinc-400">
          Are you sure you want to delete{' '}
          <span className="font-mono font-semibold text-zinc-900 dark:text-zinc-100">{key}</span>?
          This action cannot be undone.
        </p>
        <div className="mt-6 flex justify-end gap-3">
          <Button variant="secondary" onClick={() => setDeleteOpen(false)}>
            Cancel
          </Button>
          <Button
            variant="danger"
            loading={deleteMutation.isPending}
            onClick={() => deleteMutation.mutate()}
          >
            Delete
          </Button>
        </div>
      </Modal>
    </div>
  );
}

function RuleCard({ rule, index }: { rule: Rule; index: number }) {
  return (
    <div className="rounded-lg border border-zinc-100 bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-800/50">
      <div className="mb-2 flex items-center justify-between">
        <span className="text-xs font-semibold text-zinc-500">Rule {index + 1}</span>
        <span className="font-mono text-xs text-zinc-600 dark:text-zinc-400">
          {rule.rolloutPercentage}% → {rule.variant || 'default'}
        </span>
      </div>
      {rule.conditions && rule.conditions.length > 0 && (
        <div className="space-y-1">
          {rule.conditions.map((cond, i) => (
            <ConditionBadge key={i} condition={cond} />
          ))}
        </div>
      )}
    </div>
  );
}

function ConditionBadge({ condition }: { condition: Condition }) {
  return (
    <div className="inline-flex items-center gap-1 rounded-md bg-white px-2 py-1 text-xs ring-1 ring-zinc-200 dark:bg-zinc-900 dark:ring-zinc-700">
      <span className="font-medium text-zinc-700 dark:text-zinc-300">{condition.attribute}</span>
      <span className="text-zinc-400">{condition.operator}</span>
      <span className="font-mono text-zinc-600 dark:text-zinc-400">{condition.valueJson}</span>
    </div>
  );
}

function EditFlagModal({
  open,
  onClose,
  flag,
  flagKey,
}: {
  open: boolean;
  onClose: () => void;
  flag: Flag;
  flagKey: string;
}) {
  const queryClient = useQueryClient();
  const [form, setForm] = useState({
    description: flag.description || '',
    enabled: flag.enabled,
    rules: flag.rules || [],
  });

  const mutation = useMutation({
    mutationFn: (data: typeof form) => flagsAPI.update(flagKey, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['flag', flagKey] });
      queryClient.invalidateQueries({ queryKey: ['flags'] });
      onClose();
    },
  });

  const addRule = () => {
    setForm({
      ...form,
      rules: [...form.rules, { rolloutPercentage: 100, variant: '', conditions: [] }],
    });
  };

  const removeRule = (idx: number) => {
    setForm({ ...form, rules: form.rules.filter((_, i) => i !== idx) });
  };

  const updateRule = (idx: number, update: Partial<Rule>) => {
    setForm({
      ...form,
      rules: form.rules.map((r, i) => (i === idx ? { ...r, ...update } : r)),
    });
  };

  return (
    <Modal open={open} onClose={onClose} title="Edit Flag" wide>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          mutation.mutate(form);
        }}
        className="space-y-4"
      >
        <FormField label="Description" htmlFor="edit-desc">
          <Textarea
            id="edit-desc"
            rows={2}
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
          <span className="text-zinc-700 dark:text-zinc-300">Enabled</span>
        </label>

        <div>
          <div className="mb-2 flex items-center justify-between">
            <h3 className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">Rules</h3>
            <Button variant="ghost" size="sm" type="button" onClick={addRule}>
              <Plus className="h-3.5 w-3.5" />
              Add Rule
            </Button>
          </div>
          {form.rules.length === 0 ? (
            <p className="py-2 text-center text-xs text-zinc-400">No rules</p>
          ) : (
            <div className="space-y-3">
              {form.rules.map((rule, i) => (
                <div
                  key={i}
                  className="rounded-lg border border-zinc-200 p-3 dark:border-zinc-700"
                >
                  <div className="mb-2 flex items-center justify-between">
                    <span className="text-xs font-semibold text-zinc-500">Rule {i + 1}</span>
                    <button type="button" onClick={() => removeRule(i)}>
                      <X className="h-4 w-4 text-zinc-400 hover:text-red-500" />
                    </button>
                  </div>
                  <div className="grid grid-cols-2 gap-3">
                    <FormField label="Rollout %" htmlFor={`rule-pct-${i}`}>
                      <Input
                        id={`rule-pct-${i}`}
                        type="number"
                        min={0}
                        max={100}
                        value={rule.rolloutPercentage}
                        onChange={(e) =>
                          updateRule(i, { rolloutPercentage: Number(e.target.value) })
                        }
                      />
                    </FormField>
                    <FormField label="Variant" htmlFor={`rule-var-${i}`}>
                      <Input
                        id={`rule-var-${i}`}
                        value={rule.variant}
                        onChange={(e) => updateRule(i, { variant: e.target.value })}
                        placeholder="control"
                      />
                    </FormField>
                  </div>
                </div>
              ))}
            </div>
          )}
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
