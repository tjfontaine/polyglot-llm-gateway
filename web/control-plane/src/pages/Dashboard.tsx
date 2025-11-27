import { Link } from 'react-router-dom';
import {
  ArrowRight,
  BadgeCheck,
  Bot,
  Compass,
  MessageSquare,
  Route,
  Signal,
  Users,
  Zap,
} from 'lucide-react';
import { useApi, formatShortDate } from '../hooks/useApi';
import { Pill, StatusBadge } from '../components/ui';

export function Dashboard() {
  const { overview, threads, responses, loadingThreads, loadingResponses } = useApi();

  const appEntries = overview?.apps?.length
    ? overview.apps
    : overview?.frontdoors?.map((fd, idx) => ({
        name: fd.type || `frontdoor-${idx + 1}`,
        frontdoor: fd.type,
        path: fd.path,
        provider: fd.provider,
        default_model: fd.default_model,
      })) ?? [];

  const recentThreads = threads.slice(0, 3);
  const recentResponses = responses.slice(0, 3);

  return (
    <div className="space-y-6">
      {/* Welcome section */}
      <div className="rounded-2xl border border-white/10 bg-gradient-to-br from-slate-900/90 to-slate-950/90 p-6">
        <h1 className="text-2xl font-bold text-white mb-2">Welcome to Control Plane</h1>
        <p className="text-slate-400 max-w-2xl">
          Monitor your gateway configuration, routing rules, and explore stored conversations and API responses.
          All data is read-only and refreshes automatically.
        </p>
      </div>

      {/* Overview Cards Grid */}
      <div className="grid grid-cols-1 gap-5 lg:grid-cols-2">
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
                <p className="text-xs text-slate-400">Apps & providers configuration</p>
              </div>
            </div>

            <div className="mb-4 flex flex-wrap gap-3">
              <div className="rounded-xl border border-white/5 bg-slate-950/50 px-4 py-2.5">
                <div className="text-[11px] uppercase tracking-wide text-slate-500">Apps</div>
                <div className="text-xl font-semibold text-white">{appEntries.length}</div>
              </div>
              <div className="rounded-xl border border-white/5 bg-slate-950/50 px-4 py-2.5">
                <div className="text-[11px] uppercase tracking-wide text-slate-500">Providers</div>
                <div className="text-xl font-semibold text-white">{overview?.providers.length ?? 0}</div>
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
                <div className="text-xs text-slate-500">+{appEntries.length - 2} more apps</div>
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
                <p className="text-xs text-slate-400">Rules & tenants configuration</p>
              </div>
            </div>

            <div className="mb-4 flex flex-wrap gap-3">
              <div className="rounded-xl border border-white/5 bg-slate-950/50 px-4 py-2.5">
                <div className="text-[11px] uppercase tracking-wide text-slate-500">Rules</div>
                <div className="text-xl font-semibold text-white">{overview?.routing.rules.length ?? 0}</div>
              </div>
              <div className="rounded-xl border border-white/5 bg-slate-950/50 px-4 py-2.5">
                <div className="text-[11px] uppercase tracking-wide text-slate-500">Tenants</div>
                <div className="text-xl font-semibold text-white">{overview?.tenants.length ?? 0}</div>
              </div>
              <div className="rounded-xl border border-white/5 bg-slate-950/50 px-4 py-2.5">
                <div className="text-[11px] uppercase tracking-wide text-slate-500">Default</div>
                <div className="text-sm font-semibold text-white truncate max-w-[100px]">
                  {overview?.routing.default_provider || '—'}
                </div>
              </div>
            </div>

            <div className="space-y-2">
              {(overview?.routing.rules ?? []).slice(0, 2).map((rule, idx) => (
                <div
                  key={idx}
                  className="flex items-center justify-between rounded-lg border border-white/5 bg-white/5 px-3 py-2"
                >
                  <div className="text-sm text-white">→ {rule.provider}</div>
                  <div className="text-xs text-slate-400">
                    {rule.model_prefix && `prefix: ${rule.model_prefix}`}
                    {rule.model_exact && `exact: ${rule.model_exact}`}
                  </div>
                </div>
              ))}
              {(overview?.routing.rules.length ?? 0) > 2 && (
                <div className="text-xs text-slate-500">+{(overview?.routing.rules.length ?? 0) - 2} more rules</div>
              )}
            </div>

            <div className="mt-4 flex items-center justify-end text-sm text-slate-400 group-hover:text-emerald-200 transition-colors">
              <span>View routing</span>
              <ArrowRight size={16} className="ml-1 transition-transform group-hover:translate-x-1" />
            </div>
          </div>
        </Link>

        {/* Conversations Card */}
        <Link
          to="/admin/conversations"
          className="group relative rounded-2xl border border-white/10 bg-slate-900/70 p-5 shadow-[0_18px_40px_rgba(0,0,0,0.35)] transition-all duration-300 hover:border-sky-400/30 hover:shadow-[0_24px_50px_rgba(14,165,233,0.1)]"
        >
          <div className="absolute inset-0 rounded-2xl bg-gradient-to-br from-sky-500/[0.03] to-transparent opacity-0 transition-opacity group-hover:opacity-100" />
          <div className="relative">
            <div className="mb-4 flex items-center gap-3">
              <div className="rounded-xl bg-sky-500/10 p-3 text-sky-300 border border-sky-500/20">
                <MessageSquare size={24} />
              </div>
              <div>
                <h3 className="text-lg font-semibold text-white">Conversations</h3>
                <p className="text-xs text-slate-400">Chat threads & messages</p>
              </div>
            </div>

            <div className="mb-4 flex flex-wrap gap-3">
              <div className="rounded-xl border border-white/5 bg-slate-950/50 px-4 py-2.5">
                <div className="text-[11px] uppercase tracking-wide text-slate-500">Threads</div>
                <div className="text-xl font-semibold text-white">
                  {loadingThreads ? '...' : threads.length}
                </div>
              </div>
              {!overview?.storage.enabled && (
                <Pill icon={Zap} label="Storage disabled" tone="amber" />
              )}
            </div>

            {overview?.storage.enabled && (
              <div className="space-y-2">
                {recentThreads.map((thread) => (
                  <div
                    key={thread.id}
                    className="flex items-center justify-between rounded-lg border border-white/5 bg-white/5 px-3 py-2"
                  >
                    <div className="flex items-center gap-2 min-w-0">
                      <Users size={14} className="text-sky-300 flex-shrink-0" />
                      <span className="text-sm font-medium text-white truncate">
                        {thread.metadata?.title || thread.id.slice(0, 12)}
                      </span>
                    </div>
                    <span className="text-xs text-slate-400 flex-shrink-0 ml-2">
                      {thread.message_count} msg
                    </span>
                  </div>
                ))}
                {threads.length === 0 && !loadingThreads && (
                  <div className="text-sm text-slate-500">No conversations yet</div>
                )}
                {threads.length > 3 && (
                  <div className="text-xs text-slate-500">+{threads.length - 3} more threads</div>
                )}
              </div>
            )}

            <div className="mt-4 flex items-center justify-end text-sm text-slate-400 group-hover:text-sky-200 transition-colors">
              <span>View conversations</span>
              <ArrowRight size={16} className="ml-1 transition-transform group-hover:translate-x-1" />
            </div>
          </div>
        </Link>

        {/* Responses Card */}
        <Link
          to="/admin/responses"
          className="group relative rounded-2xl border border-white/10 bg-slate-900/70 p-5 shadow-[0_18px_40px_rgba(0,0,0,0.35)] transition-all duration-300 hover:border-rose-400/30 hover:shadow-[0_24px_50px_rgba(244,63,94,0.1)]"
        >
          <div className="absolute inset-0 rounded-2xl bg-gradient-to-br from-rose-500/[0.03] to-transparent opacity-0 transition-opacity group-hover:opacity-100" />
          <div className="relative">
            <div className="mb-4 flex items-center gap-3">
              <div className="rounded-xl bg-rose-500/10 p-3 text-rose-300 border border-rose-500/20">
                <Bot size={24} />
              </div>
              <div>
                <h3 className="text-lg font-semibold text-white">Responses API</h3>
                <p className="text-xs text-slate-400">OpenAI Responses records</p>
              </div>
            </div>

            <div className="mb-4 flex flex-wrap gap-3">
              <div className="rounded-xl border border-white/5 bg-slate-950/50 px-4 py-2.5">
                <div className="text-[11px] uppercase tracking-wide text-slate-500">Records</div>
                <div className="text-xl font-semibold text-white">
                  {loadingResponses ? '...' : responses.length}
                </div>
              </div>
              {(overview?.apps ?? []).some((app) => app?.enable_responses) && (
                <Pill icon={BadgeCheck} label="API enabled" tone="emerald" />
              )}
            </div>

            {overview?.storage.enabled && (
              <div className="space-y-2">
                {recentResponses.map((response) => (
                  <div
                    key={response.id}
                    className="flex items-center justify-between rounded-lg border border-white/5 bg-white/5 px-3 py-2"
                  >
                    <div className="flex items-center gap-2 min-w-0">
                      <span className="text-sm font-medium text-white truncate">
                        {response.id.slice(0, 16)}...
                      </span>
                    </div>
                    <StatusBadge status={response.status} />
                  </div>
                ))}
                {responses.length === 0 && !loadingResponses && (
                  <div className="text-sm text-slate-500">No responses yet</div>
                )}
                {responses.length > 3 && (
                  <div className="text-xs text-slate-500">+{responses.length - 3} more responses</div>
                )}
              </div>
            )}

            <div className="mt-4 flex items-center justify-end text-sm text-slate-400 group-hover:text-rose-200 transition-colors">
              <span>View responses</span>
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
              label={`Default provider: ${overview.routing.default_provider || 'none'}`}
              tone="slate"
            />
            {(overview.providers ?? []).filter((p) => p.enable_passthrough).length > 0 && (
              <Pill
                icon={Zap}
                label={`${(overview.providers ?? []).filter((p) => p.enable_passthrough).length} passthrough provider(s)`}
                tone="amber"
              />
            )}
          </div>
        </div>
      )}
    </div>
  );
}
