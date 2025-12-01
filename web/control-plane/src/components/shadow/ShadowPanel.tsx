import { useState, useMemo } from 'react';
import {
    AlertTriangle,
    CheckCircle,
    Ghost,
    RefreshCcw,
} from 'lucide-react';
import { useShadowResults } from '../../gql/hooks';
import type { Interaction } from '../../gql/graphql';
import { ShadowSummary } from './ShadowSummary';
import { ShadowComparison } from './ShadowComparison';
import { EmptyState, LoadingState } from '../ui';

interface ShadowPanelProps {
    interactionId: string;
    primary?: Interaction;
}

export function ShadowPanel({ interactionId, primary }: ShadowPanelProps) {
    const { shadowResults, loading, error, refresh } = useShadowResults(interactionId);
    const shadows = useMemo(() => shadowResults?.shadows ?? [], [shadowResults?.shadows]);
    const [selectedShadowId, setSelectedShadowId] = useState<string | null>(null);

    // Derive selectedShadow from ID - auto-select first if none selected
    const selectedShadow = useMemo(() => {
        if (shadows.length === 0) return null;
        if (selectedShadowId) {
            const found = shadows.find(s => s.id === selectedShadowId);
            if (found) return found;
        }
        return shadows[0];
    }, [shadows, selectedShadowId]);

    if (loading) {
        return <LoadingState message="Loading shadow results..." />;
    }

    if (error) {
        return (
            <div className="p-4">
                <div className="flex min-h-[240px] flex-col items-center justify-center gap-3 text-slate-400">
                    <AlertTriangle size={40} className="text-red-500/60" />
                    <div className="text-center">
                        <div className="text-sm font-medium">Unable to load shadows</div>
                        <div className="mt-1 text-xs text-slate-500">{error}</div>
                    </div>
                    <button
                        onClick={refresh}
                        className="mt-2 inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-violet-500/20 text-violet-200 hover:bg-violet-500/30 transition-colors"
                    >
                        <RefreshCcw size={14} />
                        Retry
                    </button>
                </div>
            </div>
        );
    }

    if (shadows.length === 0) {
        return (
            <div className="p-4">
                <EmptyState
                    icon={Ghost}
                    title="No shadow results"
                    description="Shadow mode was not enabled for this request, or no shadow providers were configured."
                />
            </div>
        );
    }

    const hasDivergences = shadows.some(s => s.divergences && s.divergences.length > 0);
    const hasErrors = shadows.some(s => !!s.error);

    return (
        <div className="flex flex-col h-full">
            {/* Summary header */}
            <div className="border-b border-white/10 px-5 py-4">
                <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                        <div className={`rounded-lg p-2 ${hasErrors
                            ? 'bg-red-500/10 text-red-300'
                            : hasDivergences
                                ? 'bg-amber-500/10 text-amber-300'
                                : 'bg-emerald-500/10 text-emerald-300'
                            }`}>
                            <Ghost size={20} />
                        </div>
                        <div>
                            <div className="text-sm font-semibold text-white">
                                Shadow Results
                            </div>
                            <div className="text-xs text-slate-400 mt-0.5">
                                {shadows.length} shadow provider{shadows.length !== 1 ? 's' : ''} executed
                            </div>
                        </div>
                    </div>

                    <div className="flex items-center gap-2">
                        {hasErrors && (
                            <span className="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-red-500/10 text-xs text-red-300">
                                <AlertTriangle size={12} />
                                {shadows.filter(s => !!s.error).length} error{shadows.filter(s => !!s.error).length !== 1 ? 's' : ''}
                            </span>
                        )}
                        {hasDivergences && (
                            <span className="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-amber-500/10 text-xs text-amber-300">
                                <AlertTriangle size={12} />
                                Divergences detected
                            </span>
                        )}
                        {!hasErrors && !hasDivergences && (
                            <span className="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-emerald-500/10 text-xs text-emerald-300">
                                <CheckCircle size={12} />
                                All shadows match
                            </span>
                        )}
                    </div>
                </div>
            </div>

            {/* Main content */}
            <div className="flex-1 flex overflow-hidden">
                {/* Shadow list sidebar */}
                <div className="w-80 border-r border-white/10 overflow-y-auto p-4 space-y-3">
                    {shadows.map(shadow => (
                        <ShadowSummary
                            key={shadow.id}
                            shadow={shadow}
                            onClick={() => setSelectedShadowId(shadow.id)}
                            selected={selectedShadow?.id === shadow.id}
                        />
                    ))}
                </div>

                {/* Comparison panel */}
                <div className="flex-1 overflow-y-auto p-5">
                    {selectedShadow ? (
                        <ShadowComparison shadow={selectedShadow} primary={primary} />
                    ) : (
                        <EmptyState
                            icon={Ghost}
                            title="Select a shadow"
                            description="Choose a shadow provider from the list to view comparison details"
                        />
                    )}
                </div>
            </div>
        </div>
    );
}
