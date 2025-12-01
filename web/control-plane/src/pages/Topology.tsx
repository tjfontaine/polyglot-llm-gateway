import {
  BadgeCheck,
  Bot,
  Compass,
  RefreshCcw,
  Route,
  ServerCog,
  Signal,
  Zap,
} from 'lucide-react';
import { useOverview } from '../gql/hooks';
import { PageHeader, Pill, Section, EmptyState } from '../components/ui';

export function Topology() {
  const { overview, refresh: refreshOverview } = useOverview();

  const appEntries = overview?.apps?.length
    ? overview.apps
    : overview?.frontdoors?.map((fd, idx) => ({
      name: fd.type || `frontdoor-${idx + 1}`,
      frontdoor: fd.type,
      path: fd.path,
      provider: fd.provider,
      defaultModel: fd.defaultModel,
      enableResponses: false,
      modelRouting: { prefixProviders: {} as Record<string, string>, rewrites: [] },
    })) ?? [];

  return (
    <div className="space-y-6">
      <PageHeader
        title="Topology"
        subtitle="Applications and provider configurations"
        icon={Compass}
        iconColor="text-amber-300"
        actions={
          <button
            type="button"
            onClick={() => refreshOverview()}
            className="inline-flex items-center gap-2 rounded-xl border border-white/15 bg-slate-800/50 px-4 py-2 text-sm text-slate-200 transition-colors hover:border-white/30 hover:text-white"
          >
            <RefreshCcw size={16} />
            Refresh
          </button>
        }
      />

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* Apps Section */}
        <Section>
          <div className="mb-4 flex items-center gap-3">
            <Signal size={20} className="text-emerald-300" />
            <div>
              <h2 className="text-lg font-semibold text-white">Applications</h2>
              <p className="text-xs text-slate-400">Configured frontdoor endpoints</p>
            </div>
            <div className="ml-auto">
              <Pill icon={Signal} label={`${appEntries.length} app(s)`} />
            </div>
          </div>

          <div className="space-y-3">
            {appEntries.map((app) => (
              <div
                key={`${app.frontdoor || app.name}-${app.path}`}
                className="rounded-2xl border border-white/10 bg-slate-950/60 p-4 transition-colors hover:border-white/20"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="flex items-center gap-3">
                    <div className="rounded-lg bg-emerald-500/10 p-2 text-emerald-300">
                      <Signal size={18} />
                    </div>
                    <div>
                      <div className="text-base font-semibold text-white">
                        {app.name || app.frontdoor || 'App'}
                      </div>
                      <div className="text-xs text-amber-200">{app.path}</div>
                    </div>
                  </div>
                  {app.enableResponses && (
                    <span className="rounded-md bg-emerald-500/20 px-2.5 py-1 text-xs text-emerald-100 border border-emerald-500/30">
                      <Bot size={12} className="inline mr-1" />
                      Responses API
                    </span>
                  )}
                </div>

                <div className="mt-4 flex flex-wrap gap-2">
                  {app.frontdoor && (
                    <span className="rounded-md bg-slate-800/80 px-2.5 py-1 text-xs text-slate-200">
                      frontdoor: {app.frontdoor}
                    </span>
                  )}
                  {app.provider && (
                    <span className="rounded-md bg-slate-800/80 px-2.5 py-1 text-xs text-slate-200">
                      provider: {app.provider}
                    </span>
                  )}
                  {app.defaultModel && (
                    <span className="rounded-md bg-slate-800/80 px-2.5 py-1 text-xs text-slate-200">
                      default model: {app.defaultModel}
                    </span>
                  )}
                </div>

                {/* Model Routing */}
                {(Object.keys(app.modelRouting?.prefixProviders ?? {}).length > 0 ||
                  (app.modelRouting?.rewrites?.length ?? 0) > 0) && (
                    <div className="mt-4 rounded-xl border border-white/5 bg-slate-900/50 p-3">
                      <div className="text-xs font-medium text-slate-300 mb-2">Model Routing</div>
                      <div className="space-y-1.5">
                        {Object.entries(app.modelRouting?.prefixProviders ?? {}).map(
                          ([prefix, provider]) => (
                            <div
                              key={`${app.path}-prefix-${prefix}`}
                              className="flex items-center gap-2 text-xs"
                            >
                              <Route size={12} className="text-emerald-200" />
                              <code className="rounded-sm bg-slate-800/60 px-1.5 py-0.5 text-emerald-100">
                                {prefix}*
                              </code>
                              <span className="text-slate-500">→</span>
                              <span className="text-white">{String(provider)}</span>
                            </div>
                          )
                        )}
                        {(app.modelRouting?.rewrites ?? []).map((rewrite, idx) => (
                          <div
                            key={`${app.path}-rewrite-${idx}`}
                            className="flex items-center gap-2 text-xs"
                          >
                            <RefreshCcw size={12} className="text-amber-200" />
                            <code className="rounded-sm bg-slate-800/60 px-1.5 py-0.5 text-amber-100">
                              {rewrite.modelExact ||
                                (rewrite.modelPrefix ? `${rewrite.modelPrefix}*` : '')}
                            </code>
                            <span className="text-slate-500">→</span>
                            <span className="text-white">
                              {rewrite.provider}:{rewrite.model}
                            </span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
              </div>
            ))}
            {appEntries.length === 0 && (
              <EmptyState
                icon={Signal}
                title="No applications configured"
                description="Configure apps in your gateway config file"
              />
            )}
          </div>
        </Section>

        {/* Providers Section */}
        <Section>
          <div className="mb-4 flex items-center gap-3">
            <BadgeCheck size={20} className="text-amber-300" />
            <div>
              <h2 className="text-lg font-semibold text-white">Providers</h2>
              <p className="text-xs text-slate-400">Connected LLM backends</p>
            </div>
            <div className="ml-auto">
              <Pill icon={BadgeCheck} label={`${overview?.providers?.length ?? 0} provider(s)`} />
            </div>
          </div>

          <div className="space-y-3">
            {(overview?.providers ?? []).map((p) => (
              <div
                key={`${p.name}-${p.type}`}
                className="rounded-2xl border border-white/10 bg-slate-950/60 p-4 transition-colors hover:border-white/20"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="flex items-center gap-3">
                    <div className="rounded-lg bg-amber-500/10 p-2 text-amber-300">
                      <ServerCog size={18} />
                    </div>
                    <div>
                      <div className="text-base font-semibold text-white">{p.name}</div>
                      <div className="text-xs text-emerald-200">{p.type}</div>
                    </div>
                  </div>
                </div>

                <div className="mt-4 flex flex-wrap gap-2">
                  {p.baseUrl && (
                    <span className="rounded-md bg-slate-800/80 px-2.5 py-1 text-xs text-slate-200">
                      base: {p.baseUrl}
                    </span>
                  )}
                  {p.supportsResponses && (
                    <span className="rounded-md bg-emerald-500/20 px-2.5 py-1 text-xs text-emerald-100 border border-emerald-500/30">
                      <Bot size={12} className="inline mr-1" />
                      Responses ready
                    </span>
                  )}
                  {p.enablePassthrough && (
                    <span className="rounded-md bg-amber-500/20 px-2.5 py-1 text-xs text-amber-100 border border-amber-500/30">
                      <Zap size={12} className="inline mr-1" />
                      Passthrough
                    </span>
                  )}
                </div>

                {/* Provider type details */}
                <div className="mt-4 rounded-xl border border-white/5 bg-slate-900/50 p-3">
                  <div className="text-xs text-slate-400">
                    <span className="font-medium text-slate-300">API Type:</span>{' '}
                    {p.type === 'openai' && 'OpenAI-compatible API'}
                    {p.type === 'anthropic' && 'Anthropic Claude API'}
                    {!['openai', 'anthropic'].includes(p.type) && p.type}
                  </div>
                </div>
              </div>
            ))}
            {(overview?.providers?.length ?? 0) === 0 && (
              <EmptyState
                icon={BadgeCheck}
                title="No providers configured"
                description="Add providers in your gateway config file"
              />
            )}
          </div>
        </Section>
      </div>

      {/* Summary Stats */}
      <div className="rounded-2xl border border-white/10 bg-slate-900/60 p-5">
        <h2 className="text-sm font-semibold text-white mb-4">Topology Summary</h2>
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
          <div className="rounded-xl border border-white/5 bg-slate-950/50 p-4 text-center">
            <div className="text-2xl font-bold text-white">{appEntries.length}</div>
            <div className="text-xs text-slate-400 mt-1">Applications</div>
          </div>
          <div className="rounded-xl border border-white/5 bg-slate-950/50 p-4 text-center">
            <div className="text-2xl font-bold text-white">{overview?.providers?.length ?? 0}</div>
            <div className="text-xs text-slate-400 mt-1">Providers</div>
          </div>
          <div className="rounded-xl border border-white/5 bg-slate-950/50 p-4 text-center">
            <div className="text-2xl font-bold text-white">
              {appEntries.filter((a) => a.enableResponses).length}
            </div>
            <div className="text-xs text-slate-400 mt-1">Responses-enabled</div>
          </div>
          <div className="rounded-xl border border-white/5 bg-slate-950/50 p-4 text-center">
            <div className="text-2xl font-bold text-white">
              {(overview?.providers ?? []).filter((p) => p.enablePassthrough).length}
            </div>
            <div className="text-xs text-slate-400 mt-1">Passthrough</div>
          </div>
        </div>
      </div>
    </div>
  );
}
