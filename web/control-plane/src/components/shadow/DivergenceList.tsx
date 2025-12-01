import {
    AlertCircle,
    AlertTriangle,
    Diff,
    FileQuestion,
    List,
    Minus,
    Plus,
} from 'lucide-react';
import type { Divergence } from '../../gql/graphql';

interface DivergenceListProps {
    divergences: Divergence[];
}

// Map GraphQL enum values to display config
const divergenceTypeConfig: Record<string, {
    label: string;
    icon: typeof AlertTriangle;
    colorClass: string;
}> = {
    MISSING_FIELD: {
        label: 'Missing Fields',
        icon: Minus,
        colorClass: 'text-red-300 bg-red-500/10 border-red-500/30',
    },
    EXTRA_FIELD: {
        label: 'Extra Fields',
        icon: Plus,
        colorClass: 'text-blue-300 bg-blue-500/10 border-blue-500/30',
    },
    TYPE_MISMATCH: {
        label: 'Type Mismatches',
        icon: Diff,
        colorClass: 'text-amber-300 bg-amber-500/10 border-amber-500/30',
    },
    ARRAY_LENGTH: {
        label: 'Array Length Differences',
        icon: List,
        colorClass: 'text-violet-300 bg-violet-500/10 border-violet-500/30',
    },
    NULL_MISMATCH: {
        label: 'Null/Non-null Mismatches',
        icon: FileQuestion,
        colorClass: 'text-cyan-300 bg-cyan-500/10 border-cyan-500/30',
    },
};

function groupBy<T>(array: T[], key: (item: T) => string): Record<string, T[]> {
    return array.reduce((groups, item) => {
        const group = key(item);
        groups[group] = groups[group] ?? [];
        groups[group].push(item);
        return groups;
    }, {} as Record<string, T[]>);
}

export function DivergenceList({ divergences }: DivergenceListProps) {
    if (!divergences || divergences.length === 0) {
        return (
            <div className="flex items-center gap-2 text-sm text-emerald-300 px-3 py-2 rounded-lg bg-emerald-500/10 border border-emerald-500/20">
                <AlertCircle size={16} />
                No structural divergences detected
            </div>
        );
    }

    const grouped = groupBy(divergences, d => d.type);

    return (
        <div className="space-y-4">
            <div className="flex items-center gap-2 text-sm text-amber-300 px-3 py-2 rounded-lg bg-amber-500/10 border border-amber-500/20">
                <AlertTriangle size={16} />
                {divergences.length} structural divergence{divergences.length !== 1 ? 's' : ''} detected
            </div>

            {Object.entries(grouped).map(([type, items]) => {
                const config = divergenceTypeConfig[type];
                if (!config) return null;

                const Icon = config.icon;

                return (
                    <div key={type} className={`rounded-xl border ${config.colorClass} p-4`}>
                        <div className="flex items-center gap-2 mb-3">
                            <Icon size={16} />
                            <span className="font-medium text-sm">{config.label}</span>
                            <span className="px-1.5 py-0.5 rounded-sm text-xs bg-black/20">{items.length}</span>
                        </div>

                        <ul className="space-y-2">
                            {items.map((d, idx) => (
                                <li key={idx} className="text-sm">
                                    <div className="flex items-start gap-2">
                                        <code className="px-2 py-0.5 rounded-sm bg-black/30 text-xs font-mono text-white">
                                            {d.path}
                                        </code>
                                    </div>
                                    <div className="text-xs text-slate-300 mt-1 ml-1">
                                        {d.description}
                                    </div>
                                    {(d.primary !== undefined || d.shadow !== undefined) && (
                                        <div className="grid grid-cols-2 gap-2 mt-2 text-xs">
                                            {d.primary !== undefined && (
                                                <div className="rounded-sm bg-black/20 p-2">
                                                    <div className="text-slate-400 mb-1">Primary:</div>
                                                    <code className="text-white font-mono">
                                                        {JSON.stringify(d.primary)}
                                                    </code>
                                                </div>
                                            )}
                                            {d.shadow !== undefined && (
                                                <div className="rounded-sm bg-black/20 p-2">
                                                    <div className="text-slate-400 mb-1">Shadow:</div>
                                                    <code className="text-white font-mono">
                                                        {JSON.stringify(d.shadow)}
                                                    </code>
                                                </div>
                                            )}
                                        </div>
                                    )}
                                </li>
                            ))}
                        </ul>
                    </div>
                );
            })}
        </div>
    );
}
