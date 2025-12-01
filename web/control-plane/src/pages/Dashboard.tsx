import { Link } from 'react-router-dom';
import {
  ArrowRight,
  BadgeCheck,
  Bot,
  Compass,
  Database,
  MessageSquare,
  Route,
  Signal,
  Zap,
} from 'lucide-react';
import { useOverview, useInteractions } from '../gql/hooks';
import { Pill, StatusBadge } from '../components/ui';

export function Dashboard() {
  const { overview } = useOverview();
  const { interactions, total: interactionsTotal, loading: loadingInteractions } = useInteractions({ limit: 10 });

  const appEntries = overview?.apps?.length
    ? overview.apps
    : overview?.frontdoors?.map((fd, idx) => ({
      name: fd.type || `frontdoor-${idx + 1}`,
      frontdoor: fd.type,
      path: fd.path,
      provider: fd.provider,
      defaultModel: fd.defaultModel,
    })) ?? [];

  const recentInteractions = interactions.slice(0, 4);
  const conversationCount = interactions.filter(i => i.type === 'conversation').length;
  const responseCount = interactions.filter(i => i.type === 'response').length;

  return (
    <div className="space-y-6">
      {/* Welcome section */}
      <div className="rounded-2xl border border-white/10 bg-gradient-to-br from-slate-900/90 to-slate-950/90 p-6">
        <h1 className="text-2xl font-bold text-white mb-2">Welcome to Control Plane</h1>
        <p className="text-slate-400 max-w-2xl">
          Monitor your gateway configuration, routing rules, and explore stored interactions.
          All data is read-only and refreshes automatically.
        </p>
      </div>

      {/* Overview Cards Grid */}
      <div className="grid grid-cols-1 gap-5 lg:grid-cols-3">
        {/* Topology Card */}
        <Link
          to="/admin/topology"
          className="group relative rounded-2xl border border-white/10 bg-slate-900/70 p-5 shadow-[0_18px_40px_rgba(0,0,0,0.35)] transition-all duration-300 hover:border-amber-400/30 hover:shadow-[0_24px_50px_rgba(251,191,36,0.1)]"
        >
          <div className="absolute inset-0 rounded-2xl bg-gradient-to-br from-amber-500/[0.03] to-transparent opacity-0 transition-opacity group-hover:opacity-100" />
          <div className="relative">
            <div className="mb-4 flex items-center gap-3">
              <div className="rounded-xl bg-amber-500/10 p-3 text-amber-300 border border-amber-500/20">
                <Compass size={24} />
              </div>
              <div>
                <h3 className="text-lg font-semibold text-white">Topology</h3>
                <p className="text-xs text-slate-400">Apps & providers</p>
              </div>
            </div>

            <div className="mb-4 flex flex-wrap gap-3">
              <div className="rounded-xl border border-white/5 bg-slate-950/50 px-4 py-2.5">
                <div className="text-[11px] uppercase tracking-wide text-slate-500">Apps</div>
                <div className="text-xl font-semibold text-white">{appEntries.length}</div>
              </div>
              <div className="rounded-xl border border-white/5 bg-slate-950/50 px-4 py-2.5">
                <div className="text-[11px] uppercase tracking-wide text-slate-500">Providers</div>
                <div className="text-xl font-semibold text-white">{overview?.providers?.length ?? 0}</div>
              </div>
            </div>

            <div className="space-y-2">
              {appEntries.slice(0, 2).map((app) => (
                <div
                  key={`${app.frontdoor || app.name}-${app.path}`}
                  className="flex items-center justify-between rounded-lg border border-white/5 bg-white/5 px-3 py-2"
                >
                  <div className="flex items-center gap-2">
                    <Signal size={14} className="text-emerald-300" />
                    <span className="text-sm font-medium text-white">{app.name || app.frontdoor}</span>
                  </div>
                  <span className="text-xs text-amber-200">{app.path}</span>
                </div>
              ))}
              {appEntries.length > 2 && (
                <div className="text-xs text-slate-500">+{appEntries.length - 2} more</div>
              )}
            </div>

            <div className="mt-4 flex items-center justify-end text-sm text-slate-400 group-hover:text-amber-200 transition-colors">
              <span>View topology</span>
              <ArrowRight size={16} className="ml-1 transition-transform group-hover:translate-x-1" />
            </div>
          </div>
        </Link>

        {/* Routing Card */}
        <Link
          to="/admin/routing"
          className="group relative rounded-2xl border border-white/10 bg-slate-900/70 p-5 shadow-[0_18px_40px_rgba(0,0,0,0.35)] transition-all duration-300 hover:border-emerald-400/30 hover:shadow-[0_24px_50px_rgba(34,197,94,0.1)]"
        >
          <div className="absolute inset-0 rounded-2xl bg-gradient-to-br from-emerald-500/[0.03] to-transparent opacity-0 transition-opacity group-hover:opacity-100" />
          <div className="relative">
            <div className="mb-4 flex items-center gap-3">
              <div className="rounded-xl bg-emerald-500/10 p-3 text-emerald-300 border border-emerald-500/20">
                <Route size={24} />
              </div>
              <div>
                <h3 className="text-lg font-semibold text-white">Routing</h3>
                <p className="text-xs text-slate-400">Rules & tenants</p>
              </div>
            </div>

            <div className="mb-4 flex flex-wrap gap-3">
              <div className="rounded-xl border border-white/5 bg-slate-950/50 px-4 py-2.5">
                <div className="text-[11px] uppercase tracking-wide text-slate-500">Rules</div>
                <div className="text-xl font-semibold text-white">{overview?.routing?.rules?.length ?? 0}</div>
              </div>
              <div className="rounded-xl border border-white/5 bg-slate-950/50 px-4 py-2.5">
                <div className="text-[11px] uppercase tracking-wide text-slate-500">Tenants</div>
                <div className="text-xl font-semibold text-white">{overview?.tenants?.length ?? 0}</div>
              </div>
            </div>

            <div className="space-y-2">
              {(overview?.routing?.rules ?? []).slice(0, 2).map((rule, idx) => (
                <div
                  key={idx}
                  className="flex items-center justify-between rounded-lg border border-white/5 bg-white/5 px-3 py-2"
                >
                  <div className="text-sm text-white">→ {rule.provider}</div>
                  <div className="text-xs text-slate-400">
                    {rule.modelPrefix && `prefix: ${rule.modelPrefix}`}
                    {rule.modelExact && `exact: ${rule.modelExact}`}
                  </div>
                </div>
              ))}
              {(overview?.routing?.rules?.length ?? 0) > 2 && (
                <div className="text-xs text-slate-500">+{(overview?.routing?.rules?.length ?? 0) - 2} more</div>
              )}
            </div>

            <div className="mt-4 flex items-center justify-end text-sm text-slate-400 group-hover:text-emerald-200 transition-colors">
              <span>View routing</span>
              <ArrowRight size={16} className="ml-1 transition-transform group-hover:translate-x-1" />
            </div>
          </div>
        </Link>

        {/* Data Card - Unified Interactions */}
        <Link
          to="/admin/data"
          className="group relative rounded-2xl border border-white/10 bg-slate-900/70 p-5 shadow-[0_18px_40px_rgba(0,0,0,0.35)] transition-all duration-300 hover:border-violet-400/30 hover:shadow-[0_24px_50px_rgba(139,92,246,0.1)]"
        >
          <div className="absolute inset-0 rounded-2xl bg-gradient-to-br from-violet-500/[0.03] to-transparent opacity-0 transition-opacity group-hover:opacity-100" />
          <div className="relative">
            <div className="mb-4 flex items-center gap-3">
              <div className="rounded-xl bg-violet-500/10 p-3 text-violet-300 border border-violet-500/20">
                <Database size={24} />
              </div>
              <div>
                <h3 className="text-lg font-semibold text-white">Data</h3>
                <p className="text-xs text-slate-400">Interactions & responses</p>
              </div>
            </div>

            <div className="mb-4 flex flex-wrap gap-3">
              <div className="rounded-xl border border-white/5 bg-slate-950/50 px-4 py-2.5">
                <div className="text-[11px] uppercase tracking-wide text-slate-500">Total</div>
                <div className="text-xl font-semibold text-white">
                  {loadingInteractions ? '...' : interactionsTotal}
                </div>
              </div>
              <div className="rounded-xl border border-sky-500/20 bg-sky-500/5 px-3 py-2 flex items-center gap-2">
                <MessageSquare size={14} className="text-sky-300" />
                <span className="text-sm font-medium text-white">{conversationCount}</span>
              </div>
              <div className="rounded-xl border border-rose-500/20 bg-rose-500/5 px-3 py-2 flex items-center gap-2">
                <Bot size={14} className="text-rose-300" />
                <span className="text-sm font-medium text-white">{responseCount}</span>
              </div>
            </div>

            {overview?.storage.enabled ? (
              <div className="space-y-2">
                {recentInteractions.map((interaction) => (
                  <div
                    key={interaction.id}
                    className="flex items-center justify-between rounded-lg border border-white/5 bg-white/5 px-3 py-2"
                  >
                    <div className="flex items-center gap-2 min-w-0">
                      {interaction.type === 'conversation' ? (
                        <MessageSquare size={14} className="text-sky-300 flex-shrink-0" />
                      ) : (
                        <Bot size={14} className="text-rose-300 flex-shrink-0" />
                      )}
                      <span className="text-sm font-medium text-white truncate">
                        {interaction.type === 'conversation'
                          ? interaction.metadata?.title || interaction.id.slice(0, 12)
                          : interaction.id.slice(0, 16)}
                      </span>
                    </div>
                    {interaction.type === 'response' && interaction.status && (
                      <StatusBadge status={interaction.status} />
                    )}
                    {interaction.type === 'conversation' && interaction.messageCount && (
                      <span className="text-xs text-slate-400">{interaction.messageCount} msg</span>
                    )}
                  </div>
                ))}
                {interactions.length === 0 && !loadingInteractions && (
                  <div className="text-sm text-slate-500">No interactions yet</div>
                )}
              </div>
            ) : (
              <div className="rounded-lg border border-amber-500/20 bg-amber-500/5 px-3 py-2 text-xs text-amber-200">
                <Zap size={12} className="inline mr-1" />
                Storage disabled
              </div>
            )}

            <div className="mt-4 flex items-center justify-end text-sm text-slate-400 group-hover:text-violet-200 transition-colors">
              <span>Explore data</span>
              <ArrowRight size={16} className="ml-1 transition-transform group-hover:translate-x-1" />
            </div>
          </div>
        </Link>
      </div>

      {/* Quick Status */}
      {overview && (
        <div className="rounded-2xl border border-white/10 bg-slate-900/60 p-5">
          <h2 className="text-sm font-semibold text-white mb-4">Quick Status</h2>
          <div className="flex flex-wrap gap-3">
            <Pill
              icon={Compass}
              label={`Mode: ${overview.mode}`}
              tone={overview.mode === 'multi-tenant' ? 'amber' : 'slate'}
            />
            {overview.storage.enabled ? (
              <Pill icon={BadgeCheck} label={`Storage: ${overview.storage.type}`} tone="emerald" />
            ) : (
              <Pill icon={Zap} label="Storage disabled" tone="slate" />
            )}
            <Pill
              icon={Route}
              label={`Default provider: ${overview?.routing?.defaultProvider || 'none'}`}
              tone="slate"
            />
            {(overview?.providers ?? []).filter((p) => p.enablePassthrough).length > 0 && (
              <Pill
                icon={Zap}
                label={`${(overview?.providers ?? []).filter((p) => p.enablePassthrough).length} passthrough provider(s)`}
                tone="amber"
              />
            )}
            {(overview?.apps ?? []).some((app) => app?.enableResponses) && (
              <Pill icon={Bot} label="Responses API enabled" tone="emerald" />
            )}
          </div>
        </div>
      )}
    </div>
  );
}
