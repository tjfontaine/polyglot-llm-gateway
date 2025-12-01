import React from 'react';
import type { LucideIcon } from 'lucide-react';

// Pill badge component
interface PillProps {
  icon: LucideIcon;
  label: string;
  tone?: 'slate' | 'amber' | 'emerald' | 'rose' | 'sky';
}

const toneMap: Record<string, string> = {
  slate: 'border-slate-700/70 bg-slate-800/60 text-slate-100',
  amber: 'border-amber-400/50 bg-amber-500/15 text-amber-100',
  emerald: 'border-emerald-400/50 bg-emerald-500/15 text-emerald-100',
  rose: 'border-rose-400/50 bg-rose-500/15 text-rose-100',
  sky: 'border-sky-400/50 bg-sky-500/15 text-sky-100',
};

export const Pill = ({ icon: Icon, label, tone = 'slate' }: PillProps) => (
  <span className={`inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs font-semibold ${toneMap[tone]}`}>
    <Icon size={14} />
    {label}
  </span>
);

// Info card for stats
interface InfoCardProps {
  title: string;
  value: string;
  hint?: string;
  icon: LucideIcon;
}

export const InfoCard = ({ title, value, hint, icon: Icon }: InfoCardProps) => (
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

// Overview card for landing page
interface OverviewCardProps {
  title: string;
  subtitle: string;
  icon: LucideIcon;
  iconColor?: string;
  stats?: { label: string; value: string | number }[];
  children?: React.ReactNode;
  href?: string;
  onClick?: () => void;
}

export const OverviewCard = ({
  title,
  subtitle,
  icon: Icon,
  iconColor = 'text-amber-300',
  stats,
  children,
  href,
  onClick,
}: OverviewCardProps) => {
  const content = (
    <div className="group relative rounded-2xl border border-white/10 bg-slate-900/70 p-5 shadow-[0_18px_40px_rgba(0,0,0,0.35)] transition-all duration-300 hover:border-white/20 hover:shadow-[0_24px_50px_rgba(0,0,0,0.45)]">
      <div className="absolute inset-0 rounded-2xl bg-linear-to-br from-white/2 to-transparent opacity-0 transition-opacity group-hover:opacity-100" />
      <div className="relative">
        <div className="mb-4 flex items-center gap-3">
          <div className={`rounded-xl bg-slate-800/80 p-2.5 ${iconColor}`}>
            <Icon size={22} />
          </div>
          <div>
            <h3 className="text-lg font-semibold text-white">{title}</h3>
            <p className="text-xs text-slate-400">{subtitle}</p>
          </div>
        </div>
        {stats && stats.length > 0 && (
          <div className="mb-4 flex flex-wrap gap-3">
            {stats.map((stat, i) => (
              <div key={i} className="rounded-xl border border-white/5 bg-slate-950/50 px-3 py-2">
                <div className="text-[11px] uppercase tracking-wide text-slate-500">{stat.label}</div>
                <div className="text-lg font-semibold text-white">{stat.value}</div>
              </div>
            ))}
          </div>
        )}
        {children}
        <div className="mt-4 flex items-center justify-end text-xs text-slate-400 group-hover:text-white transition-colors">
          <span>View details →</span>
        </div>
      </div>
    </div>
  );

  if (href) {
    return (
      <a href={href} className="block cursor-pointer">
        {content}
      </a>
    );
  }

  if (onClick) {
    return (
      <button type="button" onClick={onClick} className="block w-full text-left cursor-pointer">
        {content}
      </button>
    );
  }

  return content;
};

// Section container
interface SectionProps {
  children: React.ReactNode;
  className?: string;
}

export const Section = ({ children, className = '' }: SectionProps) => (
  <section className={`rounded-3xl border border-white/10 bg-slate-900/80 p-5 ${className}`}>
    {children}
  </section>
);

// Page header
interface PageHeaderProps {
  title: string;
  subtitle?: string;
  icon: LucideIcon;
  iconColor?: string;
  actions?: React.ReactNode;
}

export const PageHeader = ({ title, subtitle, icon: Icon, iconColor = 'text-amber-300', actions }: PageHeaderProps) => (
  <div className="mb-6 flex flex-wrap items-center justify-between gap-4">
    <div className="flex items-center gap-4">
      <div className={`rounded-xl bg-slate-800/80 p-3 ${iconColor}`}>
        <Icon size={28} />
      </div>
      <div>
        <h1 className="text-2xl font-bold text-white">{title}</h1>
        {subtitle && <p className="text-sm text-slate-400">{subtitle}</p>}
      </div>
    </div>
    {actions}
  </div>
);

// Empty state
interface EmptyStateProps {
  icon: LucideIcon;
  title: string;
  description?: string;
}

export const EmptyState = ({ icon: Icon, title, description }: EmptyStateProps) => (
  <div className="flex min-h-[240px] flex-col items-center justify-center gap-3 text-slate-400">
    <Icon size={40} className="text-slate-600" />
    <div className="text-center">
      <div className="text-sm font-medium">{title}</div>
      {description && <div className="mt-1 text-xs text-slate-500">{description}</div>}
    </div>
  </div>
);

// Loading state
interface LoadingStateProps {
  message?: string;
}

export const LoadingState = ({ message = 'Loading...' }: LoadingStateProps) => (
  <div className="flex min-h-[240px] flex-col items-center justify-center gap-3 text-slate-300">
    <div className="h-8 w-8 animate-spin rounded-full border-2 border-slate-600 border-t-amber-400" />
    <div className="text-sm">{message}</div>
  </div>
);

// Status badge
interface StatusBadgeProps {
  status: string;
}

export const StatusBadge = ({ status }: StatusBadgeProps) => {
  const statusStyles: Record<string, string> = {
    completed: 'bg-emerald-500/20 text-emerald-100 border-emerald-500/30',
    failed: 'bg-red-500/20 text-red-100 border-red-500/30',
    cancelled: 'bg-slate-500/20 text-slate-200 border-slate-500/30',
    in_progress: 'bg-amber-500/20 text-amber-100 border-amber-500/30',
  };

  return (
    <span className={`text-xs rounded-full border px-2.5 py-1 font-medium ${statusStyles[status] || statusStyles.in_progress}`}>
      {status}
    </span>
  );
};
