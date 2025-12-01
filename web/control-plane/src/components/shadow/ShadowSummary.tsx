import {
    AlertTriangle,
    CheckCircle,
    Clock4,
    ServerCog,
    XCircle,
    Zap,
} from 'lucide-react';
import type { ShadowResult } from '../../gql/graphql';

interface ShadowSummaryProps {
    shadow: ShadowResult;
    onClick?: () => void;
    selected?: boolean;
}

function formatDuration(ns: number): string {
    const ms = ns / 1_000_000;
    if (ms < 1000) {
        return `${ms.toFixed(0)}ms`;
    }
    return `${(ms / 1000).toFixed(2)}s`;
}

export function ShadowSummary({ shadow, onClick, selected }: ShadowSummaryProps) {
    const hasError = !!shadow.error;
    const hasDivergences = shadow.divergences && shadow.divergences.length > 0;

    return (
        <div
            onClick={onClick}
            className={`
        rounded-xl border p-4 cursor-pointer transition-all
        ${selected
                    ? 'border-violet-500/50 bg-violet-500/10'
                    : 'border-white/10 bg-slate-900/60 hover:border-white/20 hover:bg-slate-900/80'
                }
        ${hasError ? 'border-red-500/30' : hasDivergences ? 'border-amber-500/30' : ''}
      `}
        >
            <div className="flex items-start justify-between gap-3">
                <div className="flex items-center gap-3">
                    <div className={`rounded-lg p-2 ${hasError
                        ? 'bg-red-500/10 text-red-300'
                        : hasDivergences
                            ? 'bg-amber-500/10 text-amber-300'
                            : 'bg-emerald-500/10 text-emerald-300'
                        }`}>
                        <ServerCog size={18} />
                    </div>
                    <div>
                        <div className="font-medium text-white text-sm">{shadow.providerName}</div>
                        {shadow.providerModel && (
                            <div className="text-xs text-slate-400 mt-0.5">{shadow.providerModel}</div>
                        )}
                    </div>
                </div>

                <div className="flex items-center gap-2">
                    {hasError ? (
                        <span className="flex items-center gap-1 px-2 py-1 rounded-full bg-red-500/10 text-xs text-red-300">
                            <XCircle size={12} />
                            Error
                        </span>
                    ) : hasDivergences ? (
                        <span className="flex items-center gap-1 px-2 py-1 rounded-full bg-amber-500/10 text-xs text-amber-300">
                            <AlertTriangle size={12} />
                            {shadow.divergences?.length} divergence{shadow.divergences?.length !== 1 ? 's' : ''}
                        </span>
                    ) : (
                        <span className="flex items-center gap-1 px-2 py-1 rounded-full bg-emerald-500/10 text-xs text-emerald-300">
                            <CheckCircle size={12} />
                            Match
                        </span>
                    )}
                </div>
            </div>

            {/* Metrics row */}
            <div className="flex flex-wrap gap-3 mt-3 text-xs">
                <span className="flex items-center gap-1.5 text-slate-400">
                    <Clock4 size={12} />
                    {formatDuration(shadow.durationNs)}
                </span>

                {(shadow.tokensIn !== undefined || shadow.tokensOut !== undefined) && (
                    <span className="flex items-center gap-1.5 text-slate-400">
                        <Zap size={12} />
                        {shadow.tokensIn ?? 0} in / {shadow.tokensOut ?? 0} out
                    </span>
                )}
            </div>

            {/* Error message preview */}
            {hasError && shadow.error && (
                <div className="mt-3 px-3 py-2 rounded-lg bg-red-500/10 border border-red-500/20">
                    <div className="text-xs text-red-300 font-medium">{shadow.error.type}</div>
                    <div className="text-xs text-red-200/70 mt-0.5 line-clamp-2">{shadow.error.message}</div>
                </div>
            )}
        </div>
    );
}
