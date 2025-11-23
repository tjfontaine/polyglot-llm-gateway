import React, { useEffect, useMemo, useState } from 'react';
import {
  Activity,
  BadgeCheck,
  Clock4,
  Compass,
  Database,
  GitBranch,
  Loader2,
  MessageSquare,
  RefreshCcw,
  Route,
  ServerCog,
  Shield,
  Signal,
  Sparkles,
} from 'lucide-react';
import './App.css';

const API_BASE = '/admin/api';

interface Stats {
  uptime: string;
  go_version: string;
  num_goroutine: number;
  memory: {
    alloc: number;
    total_alloc: number;
    sys: number;
    num_gc: number;
  };
}

interface ModelRewrite {
  match: string;
  provider: string;
  model: string;
}

interface ModelRouting {
  prefix_providers?: Record<string, string>;
  rewrites?: ModelRewrite[];
}

interface Overview {
  mode: string;
  storage: { enabled: boolean; type: string; path?: string };
  apps?: {
    name?: string;
    frontdoor?: string;
    path: string;
    provider?: string;
    default_model?: string;
    model_routing?: ModelRouting;
  }[];
  frontdoors?: { type: string; path: string; provider?: string; default_model?: string }[];
  providers: { name: string; type: string; base_url?: string; supports_responses: boolean }[];
  routing: { default_provider: string; rules: { model_prefix?: string; model_exact?: string; provider: string }[] };
  tenants: { id: string; name: string; provider_count: number; routing_rules: number; supports_tenant: boolean }[];
}

interface ThreadSummary {
  id: string;
  created_at: number;
  updated_at: number;
  metadata?: Record<string, string>;
  message_count: number;
}

interface ThreadMessage {
  id: string;
  role: string;
  content: string;
  created_at: number;
}

interface ThreadDetail {
  id: string;
  created_at: number;
  updated_at: number;
  metadata?: Record<string, string>;
  messages: ThreadMessage[];
}

const Pill = ({ icon: Icon, label, tone = 'slate' }: { icon: React.ElementType; label: string; tone?: 'slate' | 'amber' | 'emerald' }) => {
  const toneMap: Record<string, string> = {
    slate: 'border-slate-700/70 bg-slate-800/60 text-slate-100',
    amber: 'border-amber-400/50 bg-amber-500/15 text-amber-100',
    emerald: 'border-emerald-400/50 bg-emerald-500/15 text-emerald-100',
  };
  return (
    <span className={`inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs font-semibold ${toneMap[tone]}`}>
      <Icon size={14} />
      {label}
    </span>
  );
};

const InfoCard = ({
  title,
  value,
  hint,
  icon: Icon,
}: { title: string; value: string; hint?: string; icon: React.ElementType }) => (
  <div className="rounded-2xl border border-white/10 bg-slate-900/70 p-4 shadow-[0_18px_40px_rgba(0,0,0,0.35)]">
    <div className="flex items-center justify-between text-xs uppercase tracking-wide text-slate-400">
      <span className="flex items-center gap-2">
        <Icon size={14} />
        {title}
      </span>
      {hint && <span className="text-[11px] text-slate-500">{hint}</span>}
    </div>
    <div className="mt-3 text-2xl font-semibold text-white">{value}</div>
  </div>
);

const formatBytesToMB = (bytes?: number) => {
  if (!bytes) return '0 MB';
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
};

const formatTimestamp = (epoch?: number) => {
  if (!epoch) return '--';
  return new Date(epoch * 1000).toLocaleString();
};

const formatShortDate = (epoch?: number) => {
  if (!epoch) return '--';
  return new Date(epoch * 1000).toLocaleDateString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
};

const friendlyDuration = (raw?: string) => {
  if (!raw) return '--';
  const withoutFraction = raw.split('.')[0];
  return withoutFraction.replace('h', 'h ').replace('m', 'm ').replace('s', 's');
};

function App() {
  const [stats, setStats] = useState<Stats | null>(null);
  const [overview, setOverview] = useState<Overview | null>(null);
  const [threads, setThreads] = useState<ThreadSummary[]>([]);
  const [threadsError, setThreadsError] = useState<string | null>(null);
  const [loadingThreads, setLoadingThreads] = useState(false);
  const [selectedThread, setSelectedThread] = useState<ThreadDetail | null>(null);
  const [loadingThreadDetail, setLoadingThreadDetail] = useState(false);

  useEffect(() => {
    const fetchOverview = async () => {
      try {
        const res = await fetch(`${API_BASE}/overview`);
        if (!res.ok) throw new Error('Failed to load overview');
        const data = await res.json();
        setOverview(data);
      } catch (err) {
        console.error(err);
      }
    };

    const fetchStats = async () => {
      try {
        const res = await fetch(`${API_BASE}/stats`);
        if (!res.ok) throw new Error('Failed to load stats');
        const data = await res.json();
        setStats(data);
      } catch (err) {
        console.error(err);
      }
    };

    fetchOverview();
    fetchStats();
    const interval = setInterval(fetchStats, 8000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    if (!overview) return;
    if (!overview.storage.enabled) {
      setThreads([]);
      setThreadsError('Conversation storage is disabled, so threads are unavailable.');
      return;
    }
    const fetchThreads = async () => {
      setLoadingThreads(true);
      setThreadsError(null);
      try {
        const res = await fetch(`${API_BASE}/threads?limit=50`);
        if (!res.ok) throw new Error('Failed to load threads');
        const data = await res.json();
        setThreads(data.threads ?? []);
      } catch (err) {
        console.error(err);
        setThreadsError('Unable to load threads right now.');
      } finally {
        setLoadingThreads(false);
      }
    };
    fetchThreads();
  }, [overview]);

  const openThread = async (threadId: string) => {
    setLoadingThreadDetail(true);
    try {
      const res = await fetch(`${API_BASE}/threads/${threadId}`);
      if (!res.ok) throw new Error('Failed to load thread');
      const data = await res.json();
      setSelectedThread(data);
    } catch (err) {
      console.error(err);
      setSelectedThread(null);
    } finally {
      setLoadingThreadDetail(false);
    }
  };

  const tenantLabel = useMemo(() => {
    if (!overview) return 'Mode: pending';
    if (overview.mode === 'multi-tenant') {
      const count = overview.tenants?.length ?? 0;
      return `${overview.mode} (${count} tenant${count === 1 ? '' : 's'})`;
    }
    return 'single-tenant';
  }, [overview]);

  const appEntries = useMemo(() => {
    if (overview?.apps?.length) return overview.apps;
    if (overview?.frontdoors?.length) {
      return overview.frontdoors.map((fd, idx) => ({
        name: fd.type || `frontdoor-${idx + 1}`,
        frontdoor: fd.type,
        path: fd.path,
        provider: fd.provider,
        default_model: fd.default_model,
        model_routing: { prefix_providers: {}, rewrites: [] },
      }));
    }
    return [] as Overview['apps'];
  }, [overview]);

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50">
      <div className="pointer-events-none absolute inset-0 opacity-70" aria-hidden>
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_20%_20%,rgba(251,191,36,0.12),transparent_30%),radial-gradient(circle_at_80%_15%,rgba(34,197,94,0.12),transparent_26%),linear-gradient(145deg,rgba(5,9,15,0.95),rgba(7,10,18,0.98))]" />
        <div className="absolute inset-0 bg-[linear-gradient(rgba(255,255,255,0.04)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,0.04)_1px,transparent_1px)] bg-[size:120px_120px]" />
      </div>

      <div className="relative mx-auto flex max-w-6xl flex-col gap-8 px-4 py-8">
        <header className="rounded-3xl border border-white/10 bg-slate-900/80 p-6 shadow-[0_18px_60px_rgba(0,0,0,0.45)]">
          <div className="flex flex-col gap-6 md:flex-row md:items-start md:justify-between">
            <div className="space-y-3">
              <Pill icon={Compass} label="Polyglot gateway" tone="amber" />
              <div className="text-3xl font-semibold text-white">Control surface</div>
              <p className="max-w-2xl text-sm text-slate-300">
                Fresh, read-only console for observing routing, providers, and Responses API state. Authorization is required; this UI never exposes secrets.
              </p>
              <div className="flex flex-wrap gap-2">
                <Pill icon={Shield} label={tenantLabel} />
                <Pill
                  icon={Database}
                  label={overview?.storage.enabled ? `Storage: ${overview.storage.type || 'enabled'}` : 'Storage disabled'}
                  tone={overview?.storage.enabled ? 'emerald' : 'slate'}
                />
                <Pill icon={ServerCog} label={stats ? `Go ${stats.go_version}` : 'Runtime pending'} />
              </div>
            </div>
            <div className="flex items-start gap-2 text-xs text-slate-300">
              <Sparkles className="text-amber-300" size={16} />
              <span>Static assets now fall back to the app shell; deep links under /admin work.</span>
            </div>
          </div>
        </header>

        <section className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
          <InfoCard title="Uptime" value={friendlyDuration(stats?.uptime)} hint="process" icon={Clock4} />
          <InfoCard title="Goroutines" value={stats ? `${stats.num_goroutine}` : '--'} hint="runtime" icon={Activity} />
          <InfoCard title="Memory" value={formatBytesToMB(stats?.memory.alloc)} hint="allocated" icon={ServerCog} />
          <InfoCard title="GC cycles" value={stats ? `${stats.memory.num_gc}` : '--'} hint="since start" icon={GitBranch} />
        </section>

        <section className="grid grid-cols-1 gap-4 lg:grid-cols-[1.35fr_1fr]">
          <div className="rounded-3xl border border-white/10 bg-slate-900/80 p-5">
            <div className="mb-4 flex items-center justify-between">
              <div>
                <p className="text-xs uppercase tracking-wide text-slate-400">Topology</p>
                <h2 className="text-lg font-semibold text-white">Apps & providers</h2>
              </div>
              <Pill icon={Route} label={`Routing default: ${overview?.routing?.default_provider || '—'}`} />
            </div>
            <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
              <div className="rounded-2xl border border-white/10 bg-slate-950/60 p-4">
                <div className="mb-3 flex items-center gap-2 text-sm font-semibold text-white">
                  <Signal size={16} className="text-emerald-300" /> Apps
                </div>
                <div className="space-y-2">
                  {(appEntries ?? []).map((app) => (
                    <div
                      key={`${app?.frontdoor || app?.name || 'app'}-${app?.path}`}
                      className="rounded-xl border border-white/5 bg-white/5 px-3 py-2"
                    >
                      <div className="flex items-center justify-between text-sm text-white">
                        <div className="flex items-center gap-2">
                          <span className="font-semibold">{app?.name || app?.frontdoor || 'App'}</span>
                          {app?.frontdoor && (
                            <span className="rounded-md bg-slate-800/80 px-2 py-1 text-[11px] text-slate-200">
                              frontdoor: {app.frontdoor}
                            </span>
                          )}
                        </div>
                        <span className="text-xs text-amber-200">{app?.path}</span>
                      </div>
                      <div className="mt-1 flex flex-wrap gap-2 text-xs text-slate-400">
                        {app?.provider && <span className="rounded-md bg-slate-800/80 px-2 py-1">provider: {app.provider}</span>}
                        {app?.default_model && (
                          <span className="rounded-md bg-slate-800/80 px-2 py-1">default model: {app.default_model}</span>
                        )}
                      </div>
                      {(Object.keys(app?.model_routing?.prefix_providers ?? {}).length > 0 ||
                        (app?.model_routing?.rewrites?.length ?? 0) > 0) && (
                        <div className="mt-2 space-y-1 text-[11px] text-slate-300">
                          {Object.entries(app?.model_routing?.prefix_providers ?? {}).map(([prefix, provider]) => (
                            <div key={`${app?.path}-prefix-${prefix}`} className="flex items-center gap-2">
                              <Route size={12} className="text-emerald-200" />
                              <span className="truncate">{prefix}* → {provider}</span>
                            </div>
                          ))}
                          {(app?.model_routing?.rewrites ?? []).map((rewrite, idx) => (
                            <div key={`${app?.path}-rewrite-${idx}`} className="flex items-center gap-2">
                              <RefreshCcw size={12} className="text-amber-200" />
                              <span className="truncate">{rewrite.match} → {rewrite.provider}:{rewrite.model}</span>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  ))}
                  {(appEntries?.length ?? 0) === 0 && <p className="text-sm text-slate-500">No apps configured.</p>}
                </div>
              </div>
              <div className="rounded-2xl border border-white/10 bg-slate-950/60 p-4">
                <div className="mb-3 flex items-center gap-2 text-sm font-semibold text-white">
                  <BadgeCheck size={16} className="text-amber-300" /> Providers
                </div>
                <div className="space-y-2">
                  {(overview?.providers ?? []).map((p) => (
                    <div key={`${p.name}-${p.type}`} className="rounded-xl border border-white/5 bg-white/5 px-3 py-2">
                      <div className="flex items-center justify-between text-sm text-white">
                        <span className="font-semibold">{p.name}</span>
                        <span className="text-xs text-emerald-200">{p.type}</span>
                      </div>
                      <div className="mt-1 flex flex-wrap gap-2 text-xs text-slate-400">
                        {p.base_url && <span className="rounded-md bg-slate-800/80 px-2 py-1">custom base: {p.base_url}</span>}
                        {p.supports_responses && <span className="rounded-md bg-slate-800/80 px-2 py-1">responses API ready</span>}
                      </div>
                    </div>
                  ))}
                  {(overview?.providers?.length ?? 0) === 0 && <p className="text-sm text-slate-500">No providers configured.</p>}
                </div>
              </div>
            </div>
          </div>

          <div className="rounded-3xl border border-white/10 bg-slate-900/80 p-5">
            <div className="mb-4 flex items-center justify-between">
              <div>
                <p className="text-xs uppercase tracking-wide text-slate-400">Routing</p>
                <h2 className="text-lg font-semibold text-white">Rules & tenants</h2>
              </div>
              <Pill icon={Route} label={`${overview?.routing?.rules.length ?? 0} rule(s)`} />
            </div>
            <div className="space-y-3">
              {(overview?.routing?.rules ?? []).map((rule, idx) => (
                <div key={idx} className="rounded-xl border border-white/5 bg-white/5 px-3 py-2">
                  <div className="text-sm text-white">→ {rule.provider || overview?.routing?.default_provider || 'default'}</div>
                  <div className="mt-1 text-xs text-slate-400">
                    {rule.model_prefix && <span className="mr-2">prefix: {rule.model_prefix}</span>}
                    {rule.model_exact && <span>exact: {rule.model_exact}</span>}
                    {!rule.model_exact && !rule.model_prefix && <span>default</span>}
                  </div>
                </div>
              ))}
              {(overview?.routing?.rules?.length ?? 0) === 0 && <p className="text-sm text-slate-500">No routing rules specified.</p>}
            </div>
            <div className="mt-4 rounded-2xl border border-white/5 bg-slate-950/50 p-3">
              <div className="mb-2 flex items-center gap-2 text-sm font-semibold text-white">
                <MessageSquare size={16} className="text-emerald-300" /> Tenants
              </div>
              <div className="flex flex-wrap gap-2 text-xs text-slate-300">
                {(overview?.tenants ?? []).map((t) => (
                  <span key={t.id} className="rounded-full border border-white/10 bg-white/5 px-3 py-1">
                    {t.name || t.id} · {t.provider_count} providers
                  </span>
                ))}
                {(overview?.tenants?.length ?? 0) === 0 && <span className="text-slate-500">Single-tenant mode.</span>}
              </div>
            </div>
          </div>
        </section>

        <section className="rounded-3xl border border-white/10 bg-slate-900/80 p-5">
          <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
            <div>
              <p className="text-xs uppercase tracking-wide text-slate-400">Responses API</p>
              <h2 className="text-lg font-semibold text-white">Threads explorer</h2>
              <p className="text-sm text-slate-400">Read-only view of stored threads and message history.</p>
            </div>
            <button
              type="button"
              className="inline-flex items-center gap-2 rounded-full border border-amber-400/60 bg-amber-500/15 px-3 py-2 text-xs font-semibold text-amber-100 transition hover:border-amber-300"
              onClick={() => {
                if (threads.length > 0) {
                  openThread(threads[0].id);
                }
              }}
              disabled={!threads.length}
            >
              <RefreshCcw size={14} /> Open newest
            </button>
          </div>

          {!overview?.storage.enabled && <div className="rounded-xl border border-amber-400/40 bg-amber-500/10 p-4 text-amber-100">{threadsError}</div>}

          {overview?.storage.enabled && (
            <div className="grid grid-cols-1 gap-4 lg:grid-cols-[360px_1fr]">
              <div className="rounded-2xl border border-white/10 bg-slate-950/60">
                <div className="flex items-center justify-between border-b border-white/10 px-4 py-3">
                  <div className="flex items-center gap-2 text-sm font-semibold text-white">
                    <MessageSquare size={16} className="text-emerald-300" /> Threads
                  </div>
                  <button
                    type="button"
                    className="inline-flex items-center gap-1 rounded-md border border-white/15 px-2 py-1 text-xs text-slate-200 hover:border-white/30"
                    onClick={() => {
                      setSelectedThread(null);
                      setThreadsError(null);
                      setLoadingThreads(true);
                      fetch(`${API_BASE}/threads?limit=50`)
                        .then((res) => res.json())
                        .then((data) => setThreads(data.threads ?? []))
                        .catch(() => setThreadsError('Unable to load threads right now.'))
                        .finally(() => setLoadingThreads(false));
                    }}
                  >
                    <RefreshCcw size={14} /> Refresh
                  </button>
                </div>
                <div className="max-h-[540px] space-y-2 overflow-y-auto p-3">
                  {loadingThreads && <div className="flex items-center justify-center py-10 text-slate-400">Loading threads…</div>}
                  {!loadingThreads &&
                    threads.map((thread) => (
                      <button
                        key={thread.id}
                        onClick={() => openThread(thread.id)}
                        className={`group flex w-full flex-col gap-1 rounded-xl border px-3 py-3 text-left transition ${
                          selectedThread?.id === thread.id ? 'border-amber-300/70 bg-amber-500/10' : 'border-white/10 bg-white/5 hover:border-white/25'
                        }`}
                      >
                        <div className="flex items-center justify-between">
                          <div className="truncate text-sm font-semibold text-white">{thread.metadata?.title || thread.id}</div>
                          <span className="text-xs text-emerald-200">{thread.message_count} msg</span>
                        </div>
                        <div className="flex items-center justify-between text-[11px] text-slate-400">
                          <span>{formatShortDate(thread.updated_at)}</span>
                          {thread.metadata?.topic && <span className="rounded-md bg-slate-800/80 px-2 py-0.5 text-slate-200">{thread.metadata.topic}</span>}
                        </div>
                      </button>
                    ))}
                  {!loadingThreads && threads.length === 0 && <div className="py-10 text-center text-slate-500">No threads recorded yet.</div>}
                </div>
              </div>

              <div className="rounded-2xl border border-white/10 bg-slate-950/60">
                {!selectedThread && !loadingThreadDetail && (
                  <div className="flex h-full min-h-[360px] flex-col items-center justify-center gap-3 text-slate-400">
                    <MessageSquare size={32} className="text-emerald-300" />
                    <div className="text-sm">Select a thread to inspect messages.</div>
                  </div>
                )}
                {loadingThreadDetail && (
                  <div className="flex h-full min-h-[360px] flex-col items-center justify-center gap-2 text-slate-300">
                    <Loader2 className="animate-spin" />
                    Loading thread…
                  </div>
                )}
                {selectedThread && !loadingThreadDetail && (
                  <div className="flex h-full flex-col">
                    <div className="border-b border-white/10 px-5 py-4">
                      <div className="flex flex-col gap-1">
                        <div className="text-sm font-semibold text-white">{selectedThread.metadata?.title || selectedThread.id}</div>
                        <div className="flex flex-wrap gap-2 text-xs text-slate-400">
                          <span className="inline-flex items-center gap-1 rounded-md bg-slate-800/80 px-2 py-1">
                            <Clock4 size={12} /> {formatShortDate(selectedThread.created_at)}
                          </span>
                          <span className="inline-flex items-center gap-1 rounded-md bg-slate-800/80 px-2 py-1">
                            <ServerCog size={12} /> {selectedThread.messages.length} messages
                          </span>
                        </div>
                      </div>
                    </div>
                    <div className="flex-1 space-y-3 overflow-y-auto px-5 py-4">
                      {selectedThread.messages.map((msg) => (
                        <div key={msg.id} className="rounded-2xl border border-white/10 bg-white/5 px-4 py-3">
                          <div className="flex items-center justify-between text-xs text-slate-400">
                            <span className="inline-flex items-center gap-1 text-slate-200">
                              {msg.role === 'user' ? <Shield size={14} className="text-amber-200" /> : <ServerCog size={14} className="text-emerald-200" />}
                              {msg.role}
                            </span>
                            <span>{formatShortDate(msg.created_at)}</span>
                          </div>
                          <p className="mt-2 whitespace-pre-wrap text-sm leading-relaxed text-white">{msg.content}</p>
                        </div>
                      ))}
                      {selectedThread.messages.length === 0 && <div className="text-sm text-slate-500">No messages on this thread.</div>}
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}
        </section>
      </div>
    </div>
  );
}

export default App;
