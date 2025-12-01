import { useState } from 'react';
import {
    Clock4,
    Eye,
    ServerCog,
    Split,
    XCircle,
    Zap,
} from 'lucide-react';
import type { ShadowResult, Interaction } from '../../gql/graphql';
import { DivergenceList } from './DivergenceList';

interface ShadowComparisonProps {
    shadow: ShadowResult;
    primary?: Interaction;
}

type ViewMode = 'side-by-side' | 'shadow-only';
type Stage = 'request' | 'response';

function formatDuration(ns: number): string {
    const ms = ns / 1_000_000;
    if (ms < 1000) {
        return `${ms.toFixed(0)}ms`;
    }
    return `${(ms / 1000).toFixed(2)}s`;
}

function JsonView({ data, label, colorClass }: { data: unknown; label: string; colorClass: string }) {
    if (!data) {
        return (
            <div className="text-xs text-slate-500 italic p-4">No {label.toLowerCase()} data</div>
        );
    }

    return (
        <div className={`rounded-xl border ${colorClass} px-4 py-3`}>
            <div className="text-xs text-slate-400 mb-2 font-semibold">{label}</div>
            <pre className="whitespace-pre-wrap text-xs text-slate-300 bg-slate-950/50 p-4 rounded-xl overflow-auto max-h-[400px] font-mono">
                {JSON.stringify(data, null, 2)}
            </pre>
        </div>
    );
}

export function ShadowComparison({ shadow, primary }: ShadowComparisonProps) {
    const [viewMode, setViewMode] = useState<ViewMode>('side-by-side');
    const [stage, setStage] = useState<Stage>('response');

    const hasError = !!shadow.error;

    return (
        <div className="space-y-4">
            {/* Control bar */}
            <div className="flex flex-wrap items-center justify-between gap-3 p-3 rounded-xl bg-slate-900/60 border border-white/10">
                <div className="flex items-center gap-2">
                    <span className="text-xs text-slate-400">View:</span>
                    <div className="flex rounded-lg border border-white/10 overflow-hidden">
                        <button
                            onClick={() => setStage('request')}
                            className={`px-3 py-1.5 text-xs transition-colors ${stage === 'request'
                                ? 'bg-violet-500/20 text-violet-200'
                                : 'bg-transparent text-slate-400 hover:text-white'
                                }`}
                        >
                            Request
                        </button>
                        <button
                            onClick={() => setStage('response')}
                            className={`px-3 py-1.5 text-xs transition-colors ${stage === 'response'
                                ? 'bg-violet-500/20 text-violet-200'
                                : 'bg-transparent text-slate-400 hover:text-white'
                                }`}
                        >
                            Response
                        </button>
                    </div>
                </div>

                {primary && (
                    <div className="flex items-center gap-2">
                        <span className="text-xs text-slate-400">Mode:</span>
                        <div className="flex rounded-lg border border-white/10 overflow-hidden">
                            <button
                                onClick={() => setViewMode('side-by-side')}
                                className={`px-3 py-1.5 text-xs flex items-center gap-1.5 transition-colors ${viewMode === 'side-by-side'
                                    ? 'bg-violet-500/20 text-violet-200'
                                    : 'bg-transparent text-slate-400 hover:text-white'
                                    }`}
                            >
                                <Split size={12} />
                                Side by Side
                            </button>
                            <button
                                onClick={() => setViewMode('shadow-only')}
                                className={`px-3 py-1.5 text-xs flex items-center gap-1.5 transition-colors ${viewMode === 'shadow-only'
                                    ? 'bg-violet-500/20 text-violet-200'
                                    : 'bg-transparent text-slate-400 hover:text-white'
                                    }`}
                            >
                                <Eye size={12} />
                                Shadow Only
                            </button>
                        </div>
                    </div>
                )}
            </div>

            {/* Metrics comparison */}
            <div className="flex flex-wrap gap-4">
                {primary && (
                    <div className="flex-1 min-w-[200px] p-3 rounded-xl bg-slate-900/60 border border-white/10">
                        <div className="text-xs text-slate-400 mb-2 font-semibold">Primary</div>
                        <div className="flex flex-wrap gap-3 text-xs">
                            <span className="flex items-center gap-1.5 text-slate-300">
                                <ServerCog size={12} />
                                {primary.provider}
                            </span>
                            {primary.duration && (
                                <span className="flex items-center gap-1.5 text-slate-300">
                                    <Clock4 size={12} />
                                    {primary.duration}
                                </span>
                            )}
                            {primary.response?.usage && (
                                <span className="flex items-center gap-1.5 text-slate-300">
                                    <Zap size={12} />
                                    {primary.response.usage.inputTokens ?? 0} in / {primary.response.usage.outputTokens ?? 0} out
                                </span>
                            )}
                        </div>
                    </div>
                )}

                <div className={`flex-1 min-w-[200px] p-3 rounded-xl border ${hasError ? 'bg-red-500/5 border-red-500/30' : 'bg-emerald-500/5 border-emerald-500/30'
                    }`}>
                    <div className="text-xs text-slate-400 mb-2 font-semibold">Shadow: {shadow.providerName}</div>
                    <div className="flex flex-wrap gap-3 text-xs">
                        {shadow.providerModel && (
                            <span className="flex items-center gap-1.5 text-slate-300">
                                <ServerCog size={12} />
                                {shadow.providerModel}
                            </span>
                        )}
                        <span className="flex items-center gap-1.5 text-slate-300">
                            <Clock4 size={12} />
                            {formatDuration(shadow.durationNs)}
                        </span>
                        {(shadow.tokensIn !== undefined || shadow.tokensOut !== undefined) && (
                            <span className="flex items-center gap-1.5 text-slate-300">
                                <Zap size={12} />
                                {shadow.tokensIn ?? 0} in / {shadow.tokensOut ?? 0} out
                            </span>
                        )}
                    </div>
                </div>
            </div>

            {/* Divergences */}
            {shadow.divergences && shadow.divergences.length > 0 && (
                <DivergenceList divergences={shadow.divergences} />
            )}

            {/* Error display */}
            {hasError && shadow.error && (
                <div className="rounded-xl border border-red-500/30 bg-red-500/5 p-4">
                    <div className="flex items-center gap-2 mb-3">
                        <XCircle size={16} className="text-red-300" />
                        <span className="font-medium text-red-200">Shadow Execution Error</span>
                    </div>
                    <div className="text-sm text-red-100">{shadow.error.type}</div>
                    <div className="text-xs text-red-200/70 mt-1">{shadow.error.message}</div>
                    {shadow.error.code && (
                        <code className="mt-2 block text-xs text-red-300 font-mono">{shadow.error.code}</code>
                    )}
                </div>
            )}

            {/* Content comparison */}
            {!hasError && (
                <div className="space-y-4">
                    {stage === 'request' && (
                        <>
                            {viewMode === 'side-by-side' && primary ? (
                                <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                                    <div>
                                        <div className="text-xs text-slate-400 mb-2 font-semibold uppercase tracking-wide">Primary Request</div>
                                        <JsonView
                                            data={primary.request?.canonical}
                                            label="Canonical Request"
                                            colorClass="border-violet-500/20 bg-violet-500/5"
                                        />
                                    </div>
                                    <div>
                                        <div className="text-xs text-slate-400 mb-2 font-semibold uppercase tracking-wide">Shadow Request</div>
                                        <JsonView
                                            data={shadow.request?.canonical}
                                            label="Canonical Request"
                                            colorClass="border-amber-500/20 bg-amber-500/5"
                                        />
                                    </div>
                                </div>
                            ) : (
                                <JsonView
                                    data={shadow.request?.canonical}
                                    label="Canonical Request"
                                    colorClass="border-violet-500/20 bg-violet-500/5"
                                />
                            )}

                            {shadow.request?.providerRequest && (
                                <JsonView
                                    data={shadow.request.providerRequest}
                                    label={`Provider Request (to ${shadow.providerName})`}
                                    colorClass="border-blue-500/20 bg-blue-500/5"
                                />
                            )}
                        </>
                    )}

                    {stage === 'response' && (
                        <>
                            {viewMode === 'side-by-side' && primary ? (
                                <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                                    <div>
                                        <div className="text-xs text-slate-400 mb-2 font-semibold uppercase tracking-wide">Primary Response</div>
                                        <JsonView
                                            data={primary.response?.canonical}
                                            label="Canonical Response"
                                            colorClass="border-emerald-500/20 bg-emerald-500/5"
                                        />
                                    </div>
                                    <div>
                                        <div className="text-xs text-slate-400 mb-2 font-semibold uppercase tracking-wide">Shadow Response</div>
                                        <JsonView
                                            data={shadow.response?.canonical}
                                            label="Canonical Response"
                                            colorClass="border-amber-500/20 bg-amber-500/5"
                                        />
                                    </div>
                                </div>
                            ) : (
                                <div className="space-y-4">
                                    <JsonView
                                        data={shadow.response?.raw}
                                        label="Raw Provider Response"
                                        colorClass="border-emerald-500/20 bg-emerald-500/5"
                                    />
                                    <JsonView
                                        data={shadow.response?.canonical}
                                        label="Canonical Response"
                                        colorClass="border-violet-500/20 bg-violet-500/5"
                                    />
                                    <JsonView
                                        data={shadow.response?.clientResponse}
                                        label="Client Response (What client would receive)"
                                        colorClass="border-amber-500/20 bg-amber-500/5"
                                    />
                                </div>
                            )}
                        </>
                    )}
                </div>
            )}
        </div>
    );
}
