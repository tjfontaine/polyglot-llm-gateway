import { NavLink, Outlet } from 'react-router-dom';
import {
  Activity,
  Compass,
  Database,
  GitBranch,
  LayoutDashboard,
  Route,
  ServerCog,
  Shield,
  Sparkles,
  Zap,
  Clock4,
  Bot,
} from 'lucide-react';
import { useStats, useOverview, formatBytesToMB, friendlyDuration } from '../gql/hooks';
import { Pill, InfoCard } from './ui';
import { useMemo } from 'react';

const navItems = [
  { to: '/admin', icon: LayoutDashboard, label: 'Dashboard', exact: true },
  { to: '/admin/topology', icon: Compass, label: 'Topology' },
  { to: '/admin/routing', icon: Route, label: 'Routing' },
  { to: '/admin/data', icon: Database, label: 'Data' },
];

export function Layout() {
  const { stats } = useStats();
  const { overview } = useOverview();

  const tenantLabel = useMemo(() => {
    if (!overview) return 'Mode: pending';
    if (overview.mode === 'multi-tenant') {
      const count = overview.tenants?.length ?? 0;
      return `${overview.mode} (${count} tenant${count === 1 ? '' : 's'})`;
    }
    return 'single-tenant';
  }, [overview]);

  const hasResponsesEnabled = useMemo(() => {
    return (overview?.apps ?? []).some((app) => app?.enableResponses);
  }, [overview]);

  const hasPassthroughEnabled = useMemo(() => {
    return (overview?.providers ?? []).some((p) => p?.enablePassthrough);
  }, [overview]);

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50">
      {/* Background */}
      <div className="pointer-events-none fixed inset-0 opacity-70" aria-hidden>
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_20%_20%,rgba(251,191,36,0.12),transparent_30%),radial-gradient(circle_at_80%_15%,rgba(34,197,94,0.12),transparent_26%),linear-gradient(145deg,rgba(5,9,15,0.95),rgba(7,10,18,0.98))]" />
        <div className="absolute inset-0 bg-[linear-gradient(rgba(255,255,255,0.04)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,0.04)_1px,transparent_1px)] bg-size-[120px_120px]" />
      </div>

      <div className="relative mx-auto flex max-w-7xl flex-col gap-6 px-4 py-6">
        {/* Header */}
        <header className="rounded-3xl border border-white/10 bg-slate-900/80 p-5 shadow-[0_18px_60px_rgba(0,0,0,0.45)]">
          <div className="flex flex-col gap-5 lg:flex-row lg:items-start lg:justify-between">
            <div className="space-y-3">
              <div className="flex items-center gap-3">
                <Pill icon={Compass} label="Polyglot gateway" tone="amber" />
                <span className="text-xs text-slate-500">•</span>
                <span className="text-xs text-slate-400">Control Plane</span>
              </div>
              <div className="flex flex-wrap gap-2">
                <Pill icon={Shield} label={tenantLabel} />
                <Pill
                  icon={Database}
                  label={overview?.storage.enabled ? `Storage: ${overview.storage.type || 'enabled'}` : 'Storage disabled'}
                  tone={overview?.storage.enabled ? 'emerald' : 'slate'}
                />
                <Pill icon={ServerCog} label={stats ? `Go ${stats.goVersion}` : 'Runtime pending'} />
                {hasResponsesEnabled && <Pill icon={Bot} label="Responses API" tone="emerald" />}
                {hasPassthroughEnabled && <Pill icon={Zap} label="Passthrough" tone="amber" />}
              </div>
            </div>
            <div className="flex items-start gap-2 text-xs text-slate-400">
              <Sparkles className="text-amber-300" size={14} />
              <span>Read-only console for observing routing, providers, and stored data.</span>
            </div>
          </div>

          {/* Navigation */}
          <nav className="mt-5 flex flex-wrap gap-2 border-t border-white/10 pt-5">
            {navItems.map(({ to, icon: Icon, label, exact }) => (
              <NavLink
                key={to}
                to={to}
                end={exact}
                className={({ isActive }) =>
                  `flex items-center gap-2 rounded-xl px-4 py-2.5 text-sm font-medium transition-all ${isActive
                    ? 'bg-amber-500/20 text-amber-100 border border-amber-400/40 shadow-[0_0_20px_rgba(251,191,36,0.15)]'
                    : 'text-slate-300 hover:bg-white/5 hover:text-white border border-transparent'
                  }`
                }
              >
                <Icon size={16} />
                {label}
              </NavLink>
            ))}
          </nav>
        </header>

        {/* Quick Stats Bar */}
        <section className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <InfoCard title="Uptime" value={friendlyDuration(stats?.uptime)} hint="process" icon={Clock4} />
          <InfoCard title="Goroutines" value={stats ? `${stats.numGoroutine}` : '--'} hint="runtime" icon={Activity} />
          <InfoCard title="Memory" value={formatBytesToMB(stats?.memory?.alloc)} hint="allocated" icon={ServerCog} />
          <InfoCard title="GC cycles" value={stats ? `${stats.memory?.numGC}` : '--'} hint="since start" icon={GitBranch} />
        </section>

        {/* Main Content */}
        <main className="min-h-[60vh]">
          <Outlet />
        </main>

        {/* Footer */}
        <footer className="text-center text-xs text-slate-600 pb-4">
          Polyglot LLM Gateway Control Plane
        </footer>
      </div>
    </div>
  );
}
