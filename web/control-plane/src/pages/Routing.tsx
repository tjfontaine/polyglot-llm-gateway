import {
  ArrowRight,
  GitBranch,
  RefreshCcw,
  Route,
  Users,
} from 'lucide-react';
import { useOverview } from '../gql/hooks';
import { PageHeader, Pill, Section, EmptyState } from '../components/ui';

export function Routing() {
  const { overview, refresh: refreshOverview } = useOverview();

  return (
    <div className="space-y-6">
      <PageHeader
        title="Routing"
        subtitle="Model routing rules and tenant configuration"
        icon={Route}
        iconColor="text-emerald-300"
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

      {/* Default Provider Banner */}
      <div className="rounded-2xl border border-emerald-500/20 bg-emerald-500/5 p-5">
        <div className="flex items-center gap-4">
          <div className="rounded-xl bg-emerald-500/10 p-3 text-emerald-300">
            <Route size={24} />
          </div>
          <div>
            <div className="text-sm text-slate-400">Default Provider</div>
            <div className="text-xl font-bold text-white">
              {overview?.routing?.defaultProvider || 'Not configured'}
            </div>
            <p className="text-xs text-slate-500 mt-1">
              Requests that don't match any routing rule will be sent to this provider
            </p>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* Routing Rules Section */}
        <Section>
          <div className="mb-4 flex items-center gap-3">
            <GitBranch size={20} className="text-amber-300" />
            <div>
              <h2 className="text-lg font-semibold text-white">Routing Rules</h2>
              <p className="text-xs text-slate-400">Model-to-provider mappings</p>
            </div>
            <div className="ml-auto">
              <Pill icon={GitBranch} label={`${overview?.routing?.rules?.length ?? 0} rule(s)`} />
            </div>
          </div>

          <div className="space-y-3">
            {(overview?.routing?.rules ?? []).map((rule, idx) => (
              <div
                key={idx}
                className="rounded-2xl border border-white/10 bg-slate-950/60 p-4 transition-colors hover:border-white/20"
              >
                <div className="flex items-center gap-4">
                  <div className="shrink-0 rounded-lg bg-slate-800/80 px-3 py-2">
                    <div className="text-[10px] uppercase tracking-wide text-slate-500 mb-1">
                      Rule #{idx + 1}
                    </div>
                    <div className="text-xs font-medium text-slate-300">
                      {rule.modelPrefix && (
                        <>
                          <span className="text-slate-500">prefix:</span>{' '}
                          <code className="text-amber-200">{rule.modelPrefix}</code>
                        </>
                      )}
                      {rule.modelExact && (
                        <>
                          <span className="text-slate-500">exact:</span>{' '}
                          <code className="text-emerald-200">{rule.modelExact}</code>
                        </>
                      )}
                      {!rule.modelExact && !rule.modelPrefix && (
                        <span className="text-slate-400">default</span>
                      )}
                    </div>
                  </div>

                  <ArrowRight size={18} className="text-slate-600" />

                  <div className="flex-1 rounded-lg bg-emerald-500/10 border border-emerald-500/20 px-3 py-2">
                    <div className="text-[10px] uppercase tracking-wide text-emerald-400 mb-1">
                      Provider
                    </div>
                    <div className="text-sm font-semibold text-white">
                      {rule.provider || overview?.routing?.defaultProvider || 'default'}
                    </div>
                  </div>
                </div>

                {/* Rule explanation */}
                <div className="mt-3 text-xs text-slate-500">
                  {rule.modelPrefix && (
                    <>Models starting with <code className="text-amber-200/70">{rule.modelPrefix}</code> will be routed to <span className="text-white">{rule.provider}</span></>
                  )}
                  {rule.modelExact && (
                    <>Exact model <code className="text-emerald-200/70">{rule.modelExact}</code> will be routed to <span className="text-white">{rule.provider}</span></>
                  )}
                  {!rule.modelExact && !rule.modelPrefix && (
                    <>Default routing rule to <span className="text-white">{rule.provider}</span></>
                  )}
                </div>
              </div>
            ))}
            {(overview?.routing?.rules?.length ?? 0) === 0 && (
              <EmptyState
                icon={GitBranch}
                title="No routing rules configured"
                description="All requests use the default provider"
              />
            )}
          </div>

          {/* Add rule explanation */}
          <div className="mt-4 rounded-xl border border-white/5 bg-slate-950/30 p-4">
            <h3 className="text-sm font-medium text-slate-300 mb-2">How routing works</h3>
            <ul className="space-y-2 text-xs text-slate-500">
              <li className="flex items-start gap-2">
                <span className="text-amber-300">•</span>
                <span><strong className="text-slate-400">Prefix rules:</strong> Match models starting with a specific string (e.g., <code className="text-amber-200/70">gpt-</code> matches <code className="text-slate-400">gpt-4</code>, <code className="text-slate-400">gpt-3.5-turbo</code>)</span>
              </li>
              <li className="flex items-start gap-2">
                <span className="text-emerald-300">•</span>
                <span><strong className="text-slate-400">Exact rules:</strong> Match a specific model name exactly</span>
              </li>
              <li className="flex items-start gap-2">
                <span className="text-slate-400">•</span>
                <span>Rules are evaluated in order; first match wins</span>
              </li>
            </ul>
          </div>
        </Section>

        {/* Tenants Section */}
        <Section>
          <div className="mb-4 flex items-center gap-3">
            <Users size={20} className="text-sky-300" />
            <div>
              <h2 className="text-lg font-semibold text-white">Tenants</h2>
              <p className="text-xs text-slate-400">Multi-tenant configuration</p>
            </div>
            <div className="ml-auto">
              <Pill
                icon={Users}
                label={overview?.mode === 'multi-tenant' ? `${overview?.tenants?.length ?? 0} tenant(s)` : 'Single tenant'}
                tone={overview?.mode === 'multi-tenant' ? 'sky' : 'slate'}
              />
            </div>
          </div>

          {overview?.mode === 'single-tenant' && (
            <div className="rounded-2xl border border-white/10 bg-slate-950/60 p-6 text-center">
              <Users size={40} className="mx-auto mb-4 text-slate-600" />
              <h3 className="text-lg font-semibold text-white mb-2">Single-tenant Mode</h3>
              <p className="text-sm text-slate-400 max-w-sm mx-auto">
                The gateway is running in single-tenant mode. All requests share the same provider configuration.
              </p>
            </div>
          )}

          {overview?.mode === 'multi-tenant' && (
            <div className="space-y-3">
              {(overview.tenants ?? []).map((tenant) => (
                <div
                  key={tenant.id}
                  className="rounded-2xl border border-white/10 bg-slate-950/60 p-4 transition-colors hover:border-white/20"
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex items-center gap-3">
                      <div className="rounded-lg bg-sky-500/10 p-2 text-sky-300">
                        <Users size={18} />
                      </div>
                      <div>
                        <div className="text-base font-semibold text-white">
                          {tenant.name || tenant.id}
                        </div>
                        <div className="text-xs text-slate-500">ID: {tenant.id}</div>
                      </div>
                    </div>
                  </div>

                  <div className="mt-4 grid grid-cols-2 gap-3">
                    <div className="rounded-xl border border-white/5 bg-slate-900/50 p-3 text-center">
                      <div className="text-lg font-bold text-white">{tenant.providerCount}</div>
                      <div className="text-xs text-slate-400">Providers</div>
                    </div>
                    <div className="rounded-xl border border-white/5 bg-slate-900/50 p-3 text-center">
                      <div className="text-lg font-bold text-white">{tenant.routingRules}</div>
                      <div className="text-xs text-slate-400">Routing rules</div>
                    </div>
                  </div>

                  {tenant.supportsTenant && (
                    <div className="mt-3">
                      <span className="rounded-md bg-emerald-500/20 px-2.5 py-1 text-xs text-emerald-100 border border-emerald-500/30">
                        Custom configuration
                      </span>
                    </div>
                  )}
                </div>
              ))}
              {(overview.tenants?.length ?? 0) === 0 && (
                <EmptyState
                  icon={Users}
                  title="No tenants configured"
                  description="Add tenants in your gateway config"
                />
              )}
            </div>
          )}

          {/* Multi-tenant explanation */}
          {overview?.mode === 'multi-tenant' && (
            <div className="mt-4 rounded-xl border border-white/5 bg-slate-950/30 p-4">
              <h3 className="text-sm font-medium text-slate-300 mb-2">Multi-tenant mode</h3>
              <p className="text-xs text-slate-500">
                Each tenant can have its own provider configuration and routing rules.
                Requests are routed to the appropriate tenant based on the API key used.
              </p>
            </div>
          )}
        </Section>
      </div>

      {/* Routing Summary */}
      <div className="rounded-2xl border border-white/10 bg-slate-900/60 p-5">
        <h2 className="text-sm font-semibold text-white mb-4">Routing Summary</h2>
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
          <div className="rounded-xl border border-white/5 bg-slate-950/50 p-4 text-center">
            <div className="text-2xl font-bold text-white">{overview?.routing?.rules?.length ?? 0}</div>
            <div className="text-xs text-slate-400 mt-1">Routing Rules</div>
          </div>
          <div className="rounded-xl border border-white/5 bg-slate-950/50 p-4 text-center">
            <div className="text-2xl font-bold text-white">{overview?.tenants?.length ?? 0}</div>
            <div className="text-xs text-slate-400 mt-1">Tenants</div>
          </div>
          <div className="rounded-xl border border-white/5 bg-slate-950/50 p-4 text-center">
            <div className="text-2xl font-bold text-white">
              {(overview?.routing?.rules ?? []).filter((r) => r.modelPrefix).length}
            </div>
            <div className="text-xs text-slate-400 mt-1">Prefix Rules</div>
          </div>
          <div className="rounded-xl border border-white/5 bg-slate-950/50 p-4 text-center">
            <div className="text-2xl font-bold text-white">
              {(overview?.routing?.rules ?? []).filter((r) => r.modelExact).length}
            </div>
            <div className="text-xs text-slate-400 mt-1">Exact Rules</div>
          </div>
        </div>
      </div>
    </div>
  );
}
