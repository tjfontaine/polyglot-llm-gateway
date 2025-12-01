import { useState, Fragment } from 'react';
import {
  AlertCircle,
  ArrowDown,
  ArrowLeftRight,
  Bot,
  CheckCircle,
  Clock4,
  Compass,
  Database,
  Ghost,
  List,
  Loader2,
  MessageSquare,
  RefreshCcw,
  Route,
  ServerCog,
  Shield,
  Terminal,
  Wrench,
  Zap,
} from 'lucide-react';
import { PageHeader, Pill, EmptyState, LoadingState, StatusBadge, ShadowPanel } from '../components';
import {
  useOverview,
  useInteractions,
  useInteraction,
  useInteractionEvents,
  formatShortDate,
} from '../gql/hooks';
import type {
  Interaction,
  InteractionFilter,
} from '../gql/graphql';

// Filter by frontdoor type or status - unified model doesn't distinguish conversation/response
type FilterType = '' | 'openai' | 'anthropic' | 'responses';
type DetailTab = 'pipeline' | 'shadows' | 'timeline';

function EventTimeline({ interactionId }: { interactionId: string }) {
  const { events: eventsData, loading, error } = useInteractionEvents(interactionId);
  const events = eventsData?.events ?? [];

  if (loading) return <LoadingState message="Loading timeline..." />;
  if (error) return <EmptyState icon={AlertCircle} title="Timeline unavailable" description={error} />;
  if (!events.length) return <EmptyState icon={Clock4} title="No events" description="No audit events recorded for this interaction." />;

  return (
    <div className="space-y-3">
      {events.map(evt => (
        <div key={evt.id} className="rounded-lg border border-white/5 bg-slate-900/60 p-3">
          <div className="flex flex-wrap items-center gap-2 text-xs text-slate-300 mb-1">
            <span className="font-mono text-white">{evt.stage}</span>
            <span className="px-1.5 py-0.5 rounded-sm bg-slate-800 text-[10px] uppercase tracking-wide text-slate-300">{evt.direction}</span>
            {evt.modelRequested && <span className="px-1.5 py-0.5 rounded-sm bg-slate-800 text-[10px] text-slate-200">req: {evt.modelRequested}</span>}
            {evt.modelServed && <span className="px-1.5 py-0.5 rounded-sm bg-slate-800 text-[10px] text-slate-200">served: {evt.modelServed}</span>}
            {evt.threadKey && <span className="px-1.5 py-0.5 rounded-sm bg-slate-800 text-[10px] text-amber-200">thread: {evt.threadKey}</span>}
            {evt.previousResponseId && <span className="px-1.5 py-0.5 rounded-sm bg-slate-800 text-[10px] text-amber-100">prev: {evt.previousResponseId}</span>}
          </div>
          <div className="grid grid-cols-1 gap-2 md:grid-cols-2">
            {evt.raw && (
              <div>
                <div className="text-[10px] uppercase tracking-wide text-slate-500 mb-1">Raw</div>
                <pre className="text-xs bg-slate-950/70 border border-white/5 rounded-sm p-2 overflow-auto max-h-60">
                  {JSON.stringify(evt.raw, null, 2)}
                </pre>
              </div>
            )}
            {evt.canonical && (
              <div>
                <div className="text-[10px] uppercase tracking-wide text-slate-500 mb-1">Canonical</div>
                <pre className="text-xs bg-slate-950/70 border border-white/5 rounded-sm p-2 overflow-auto max-h-60">
                  {JSON.stringify(evt.canonical, null, 2)}
                </pre>
              </div>
            )}
          </div>
        </div>
      ))}
    </div>
  );
}

// Component for the pipeline flow content (extracted from UnifiedInteractionDetail)
function PipelineFlowContent({ interaction }: { interaction: Interaction }) {
  const requestHeaders = interaction.requestHeaders as Record<string, string> | null;

  return (
    <div className="flex-1 space-y-6 overflow-y-auto px-5 py-6 max-h-[calc(100vh-450px)]">
      {/* Request Headers */}
      {requestHeaders && Object.keys(requestHeaders).length > 0 && (
        <div className="rounded-2xl border border-violet-500/20 bg-violet-500/5 px-4 py-3">
          <div className="flex items-center gap-2 text-xs text-slate-400 mb-3">
            <List size={14} className="text-violet-300" />
            <span className="font-semibold text-violet-200">Request Headers</span>
          </div>
          <div className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-xs font-mono">
            {Object.entries(requestHeaders).map(([key, value]) => (
              <Fragment key={key}>
                <div className="text-slate-400 text-right">{key}:</div>
                <div className="text-slate-200 break-all">{value}</div>
              </Fragment>
            ))}
          </div>
        </div>
      )}

      {/* STEP 1: Client Request (Raw) */}
      {interaction.request?.raw && (
        <>
          <div className="rounded-2xl border border-amber-500/20 bg-amber-500/5 px-4 py-3">
            <div className="flex items-center gap-2 text-xs text-slate-400 mb-3">
              <span className="flex items-center justify-center w-6 h-6 rounded-full bg-amber-500/20 text-amber-200 font-bold text-[10px]">1</span>
              <Shield size={14} className="text-amber-300" />
              <span className="font-semibold text-amber-200">Client Request</span>
              <span className="text-slate-500">(Raw from {interaction.frontdoor})</span>
            </div>
            <pre className="whitespace-pre-wrap text-xs text-slate-300 bg-slate-950/50 p-4 rounded-xl overflow-auto max-h-[300px] font-mono">
              {JSON.stringify(interaction.request.raw, null, 2)}
            </pre>
          </div>
          <div className="flex items-center justify-center gap-2 py-2">
            <div className="h-8 w-0.5 bg-linear-to-b from-amber-500/40 to-violet-500/40"></div>
            <ArrowDown size={16} className="text-violet-300" />
            <span className="text-xs text-slate-400">Decode to Canonical</span>
          </div>
        </>
      )}

      {/* STEP 2: Canonical Request */}
      {interaction.request?.canonical && (
        <>
          <div className="rounded-2xl border border-violet-500/20 bg-violet-500/5 px-4 py-3">
            <div className="flex items-center gap-2 text-xs text-slate-400 mb-3">
              <span className="flex items-center justify-center w-6 h-6 rounded-full bg-violet-500/20 text-violet-200 font-bold text-[10px]">2</span>
              <ArrowLeftRight size={14} className="text-violet-300" />
              <span className="font-semibold text-violet-200">Canonical Request</span>
              <span className="text-slate-500">(Normalized format)</span>
            </div>
            <pre className="whitespace-pre-wrap text-xs text-slate-300 bg-slate-950/50 p-4 rounded-xl overflow-auto max-h-[300px] font-mono">
              {JSON.stringify(interaction.request.canonical, null, 2)}
            </pre>
            {interaction.request.unmappedFields && interaction.request.unmappedFields.length > 0 && (
              <div className="mt-3 rounded-lg border border-amber-400/30 bg-amber-500/10 px-3 py-2">
                <div className="text-xs font-medium text-amber-200 mb-1">⚠️ Unmapped Fields</div>
                <div className="flex flex-wrap gap-1">
                  {interaction.request.unmappedFields.map((field) => (
                    <span key={field} className="px-2 py-0.5 rounded-sm bg-amber-500/20 text-[10px] text-amber-200 font-mono">
                      {field}
                    </span>
                  ))}
                </div>
              </div>
            )}
          </div>
          <div className="flex items-center justify-center gap-2 py-2">
            <div className="h-8 w-0.5 bg-linear-to-b from-violet-500/40 to-blue-500/40"></div>
            <ArrowDown size={16} className="text-blue-300" />
            <span className="text-xs text-slate-400">Encode for Provider</span>
          </div>
        </>
      )}

      {/* STEP 3: Provider Request */}
      {interaction.request?.providerRequest && (
        <>
          <div className="rounded-2xl border border-blue-500/20 bg-blue-500/5 px-4 py-3">
            <div className="flex items-center gap-2 text-xs text-slate-400 mb-3">
              <span className="flex items-center justify-center w-6 h-6 rounded-full bg-blue-500/20 text-blue-200 font-bold text-[10px]">3</span>
              <ServerCog size={14} className="text-blue-300" />
              <span className="font-semibold text-blue-200">Provider Request</span>
              <span className="text-slate-500">(Sent to {interaction.provider})</span>
            </div>
            <pre className="whitespace-pre-wrap text-xs text-slate-300 bg-slate-950/50 p-4 rounded-xl overflow-auto max-h-[300px] font-mono">
              {JSON.stringify(interaction.request.providerRequest, null, 2)}
            </pre>
          </div>
          <div className="flex items-center justify-center gap-2 py-4">
            <div className="flex-1 border-t-2 border-dashed border-emerald-500/40"></div>
            <div className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-emerald-500/10 border border-emerald-500/30">
              <Zap size={14} className="text-emerald-300" />
              <span className="text-xs font-medium text-emerald-200">API Call to {interaction.provider}</span>
              <Zap size={14} className="text-emerald-300" />
            </div>
            <div className="flex-1 border-t-2 border-dashed border-emerald-500/40"></div>
          </div>
        </>
      )}

      {/* STEP 4: Provider Response */}
      {interaction.response?.raw && (
        <>
          <div className="rounded-2xl border border-emerald-500/20 bg-emerald-500/5 px-4 py-3">
            <div className="flex items-center gap-2 text-xs text-slate-400 mb-3">
              <span className="flex items-center justify-center w-6 h-6 rounded-full bg-emerald-500/20 text-emerald-200 font-bold text-[10px]">4</span>
              <ServerCog size={14} className="text-emerald-300" />
              <span className="font-semibold text-emerald-200">Provider Response</span>
              <span className="text-slate-500">(Raw from {interaction.provider})</span>
            </div>
            <pre className="whitespace-pre-wrap text-xs text-slate-300 bg-slate-950/50 p-4 rounded-xl overflow-auto max-h-[300px] font-mono">
              {JSON.stringify(interaction.response.raw, null, 2)}
            </pre>
          </div>
          <div className="flex items-center justify-center gap-2 py-2">
            <div className="h-8 w-0.5 bg-linear-to-b from-emerald-500/40 to-violet-500/40"></div>
            <ArrowDown size={16} className="text-violet-300" />
            <span className="text-xs text-slate-400">Decode to Canonical</span>
          </div>
        </>
      )}

      {/* STEP 5: Canonical Response */}
      {interaction.response?.canonical && (
        <>
          <div className="rounded-2xl border border-violet-500/20 bg-violet-500/5 px-4 py-3">
            <div className="flex items-center gap-2 text-xs text-slate-400 mb-3">
              <span className="flex items-center justify-center w-6 h-6 rounded-full bg-violet-500/20 text-violet-200 font-bold text-[10px]">5</span>
              <ArrowLeftRight size={14} className="text-violet-300" />
              <span className="font-semibold text-violet-200">Canonical Response</span>
              <span className="text-slate-500">(Normalized format)</span>
            </div>
            <pre className="whitespace-pre-wrap text-xs text-slate-300 bg-slate-950/50 p-4 rounded-xl overflow-auto max-h-[300px] font-mono">
              {JSON.stringify(interaction.response.canonical, null, 2)}
            </pre>
            {interaction.response.unmappedFields && interaction.response.unmappedFields.length > 0 && (
              <div className="mt-3 rounded-lg border border-amber-400/30 bg-amber-500/10 px-3 py-2">
                <div className="text-xs font-medium text-amber-200 mb-1">⚠️ Unmapped Fields</div>
                <div className="flex flex-wrap gap-1">
                  {interaction.response.unmappedFields.map((field) => (
                    <span key={field} className="px-2 py-0.5 rounded-sm bg-amber-500/20 text-[10px] text-amber-200 font-mono">
                      {field}
                    </span>
                  ))}
                </div>
              </div>
            )}
          </div>
          <div className="flex items-center justify-center gap-2 py-2">
            <div className="h-8 w-0.5 bg-linear-to-b from-violet-500/40 to-cyan-500/40"></div>
            <ArrowDown size={16} className="text-cyan-300" />
            <span className="text-xs text-slate-400">Encode for Client</span>
          </div>
        </>
      )}

      {/* STEP 6: Client Response */}
      {interaction.response?.clientResponse && (
        <div className="rounded-2xl border border-cyan-500/20 bg-cyan-500/5 px-4 py-3">
          <div className="flex items-center gap-2 text-xs text-slate-400 mb-3">
            <span className="flex items-center justify-center w-6 h-6 rounded-full bg-cyan-500/20 text-cyan-200 font-bold text-[10px]">6</span>
            <CheckCircle size={14} className="text-cyan-300" />
            <span className="font-semibold text-cyan-200">Client Response</span>
            <span className="text-slate-500">(Returned to client)</span>
          </div>
          <pre className="whitespace-pre-wrap text-xs text-slate-300 bg-slate-950/50 p-4 rounded-xl overflow-auto max-h-[300px] font-mono">
            {JSON.stringify(interaction.response.clientResponse, null, 2)}
          </pre>
          {interaction.response.usage && (
            <div className="mt-3 rounded-lg border border-slate-500/30 bg-slate-800/30 px-3 py-2">
              <div className="text-xs font-medium text-slate-300 mb-2">Token Usage</div>
              <div className="grid grid-cols-3 gap-4 text-xs">
                <div>
                  <div className="text-slate-500">Prompt</div>
                  <div className="text-slate-200 font-mono">{interaction.response.usage.inputTokens}</div>
                </div>
                <div>
                  <div className="text-slate-500">Completion</div>
                  <div className="text-slate-200 font-mono">{interaction.response.usage.outputTokens}</div>
                </div>
                <div>
                  <div className="text-slate-500">Total</div>
                  <div className="text-emerald-200 font-mono font-semibold">{interaction.response.usage.totalTokens}</div>
                </div>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Error State */}
      {interaction.error && (
        <div className="rounded-2xl border border-red-500/30 bg-red-500/10 px-4 py-3">
          <div className="flex items-center gap-2 text-xs text-red-300 mb-3">
            <AlertCircle size={14} />
            <span className="font-semibold">Error</span>
          </div>
          <div className="text-sm text-red-200">
            <div className="font-mono font-medium">{interaction.error.type}</div>
            {interaction.error.code && <div className="text-xs mt-1 text-red-300">Code: {interaction.error.code}</div>}
            <div className="mt-2 text-red-100">{interaction.error.message}</div>
          </div>
        </div>
      )}
    </div>
  );
}

function UnifiedInteractionDetail({ interaction }: { interaction: Interaction }) {
  const [activeTab, setActiveTab] = useState<DetailTab>('pipeline');

  return (
    <div className="flex h-full flex-col">
      {/* Detail Header */}
      <div className="border-b border-white/10 px-5 py-4">
        <div className="flex flex-col gap-3">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-violet-500/10 p-2 text-violet-300">
              <ArrowLeftRight size={20} />
            </div>
            <div className="min-w-0 flex-1">
              <div className="text-sm font-mono font-semibold text-white truncate">
                {interaction.id}
              </div>
              <div className="flex items-center gap-2 mt-1">
                <span className="rounded-md px-2 py-0.5 text-xs font-medium bg-violet-500/20 text-violet-200">
                  interaction
                </span>
                {interaction.status && (
                  <StatusBadge status={interaction.status} />
                )}
              </div>
            </div>
          </div>
          <div className="flex flex-wrap gap-2">
            <span className="inline-flex items-center gap-1.5 rounded-md bg-slate-800/80 px-2.5 py-1 text-xs text-slate-300">
              <Clock4 size={12} />
              {formatShortDate(interaction.createdAt)}
            </span>
            {interaction.servedModel && (
              <span className="inline-flex items-center gap-1.5 rounded-md bg-slate-800/80 px-2.5 py-1 text-xs text-slate-300">
                <ServerCog size={12} />
                {interaction.servedModel}
              </span>
            )}
            {interaction.frontdoor && (
              <span className="inline-flex items-center gap-1.5 rounded-md bg-slate-800/80 px-2.5 py-1 text-xs text-slate-300">
                <Route size={12} />
                fd: {interaction.frontdoor}
              </span>
            )}
            {interaction.provider && (
              <span className="inline-flex items-center gap-1.5 rounded-md bg-slate-800/80 px-2.5 py-1 text-xs text-emerald-200">
                <ServerCog size={12} />
                provider: {interaction.provider}
              </span>
            )}
            {interaction.appName && (
              <span className="inline-flex items-center gap-1.5 rounded-md bg-slate-800/80 px-2.5 py-1 text-xs text-slate-200">
                <Compass size={12} />
                app: {interaction.appName}
              </span>
            )}
            {interaction.duration && (
              <span className="inline-flex items-center gap-1.5 rounded-md bg-slate-800/80 px-2.5 py-1 text-xs text-slate-300">
                <Clock4 size={12} />
                {interaction.duration}
              </span>
            )}
          </div>
        </div>
      </div>

      {/* Tab Bar */}
      <div className="border-b border-white/10 px-5 py-2 bg-slate-900/30">
        <div className="flex items-center gap-1">
          <button
            onClick={() => setActiveTab('pipeline')}
            className={`px-4 py-2 text-sm rounded-lg transition-colors flex items-center gap-2 ${activeTab === 'pipeline'
              ? 'bg-violet-500/20 text-violet-200 border border-violet-400/30'
              : 'text-slate-400 hover:text-white hover:bg-white/5'
              }`}
          >
            <ArrowLeftRight size={14} />
            Pipeline
          </button>
          <button
            onClick={() => setActiveTab('shadows')}
            className={`px-4 py-2 text-sm rounded-lg transition-colors flex items-center gap-2 ${activeTab === 'shadows'
              ? 'bg-amber-500/20 text-amber-200 border border-amber-400/30'
              : 'text-slate-400 hover:text-white hover:bg-white/5'
              }`}
          >
            <Ghost size={14} />
            Shadows
          </button>
          <button
            onClick={() => setActiveTab('timeline')}
            className={`px-4 py-2 text-sm rounded-lg transition-colors flex items-center gap-2 ${activeTab === 'timeline'
              ? 'bg-cyan-500/20 text-cyan-200 border border-cyan-400/30'
              : 'text-slate-400 hover:text-white hover:bg-white/5'
              }`}
          >
            <Clock4 size={14} />
            Timeline
          </button>
        </div>
      </div>

      {/* Tab Content */}
      {activeTab === 'pipeline' && <PipelineFlowContent interaction={interaction} />}

      {activeTab === 'shadows' && (
        <ShadowPanel interactionId={interaction.id} primary={interaction} />
      )}

      {activeTab === 'timeline' && (
        <div className="p-5">
          <EventTimeline interactionId={interaction.id} />
        </div>
      )}
    </div>
  );
}

export function Data() {
  const { overview } = useOverview();
  const [filter, setFilter] = useState<FilterType>('');
  const [selectedId, setSelectedId] = useState<string | null>(null);

  // Build filter for interactions query
  const interactionFilter: InteractionFilter | undefined = filter
    ? { frontdoor: filter }
    : undefined;

  const { interactions, total: interactionsTotal, loading: loadingInteractions, error: interactionsError, refresh: refreshInteractions } = useInteractions({
    filter: interactionFilter,
    limit: 50,
  });

  // Fetch selected interaction detail
  const { interaction: selectedInteraction, loading: loadingDetail } = useInteraction(selectedId);

  const openInteraction = (id: string) => {
    setSelectedId(id);
  };

  const handleRefresh = () => {
    setSelectedId(null);
    refreshInteractions();
  };

  const handleFilterChange = (newFilter: FilterType) => {
    setFilter(newFilter);
    setSelectedId(null);
  };

  if (!overview?.storage.enabled) {
    return (
      <div className="space-y-6">
        <PageHeader
          title="Data Explorer"
          subtitle="LLM Interactions"
          icon={Database}
          iconColor="text-violet-300"
        />
        <div className="rounded-2xl border border-amber-400/30 bg-amber-500/10 p-8 text-center">
          <Zap size={48} className="mx-auto mb-4 text-amber-300" />
          <h2 className="text-xl font-semibold text-white mb-2">Storage Disabled</h2>
          <p className="text-slate-400 max-w-md mx-auto">
            Storage is disabled. Enable storage in your gateway configuration to view recorded interactions.
          </p>
        </div>
      </div>
    );
  }

  // Helper to extract frontdoor from metadata (which is JSON scalar)
  const getFrontdoor = (metadata: unknown): string => {
    if (metadata && typeof metadata === 'object' && 'frontdoor' in metadata) {
      return (metadata as { frontdoor?: string }).frontdoor || 'unknown';
    }
    return 'unknown';
  };

  // Count interactions by frontdoor type
  const openaiCount = interactions.filter(i => getFrontdoor(i.metadata) === 'openai').length;
  const anthropicCount = interactions.filter(i => getFrontdoor(i.metadata) === 'anthropic').length;
  const responsesCount = interactions.filter(i => getFrontdoor(i.metadata) === 'responses').length;


  return (
    <div className="space-y-6">
      <PageHeader
        title="Data Explorer"
        subtitle={`${interactionsTotal} interaction${interactionsTotal !== 1 ? 's' : ''} recorded`}
        icon={Database}
        iconColor="text-violet-300"
        actions={
          <div className="flex items-center gap-3">
            {/* Filter tabs */}
            <div className="flex items-center rounded-xl border border-white/10 bg-slate-950/60 p-1">
              <button
                type="button"
                className={`rounded-lg px-3 py-1.5 text-xs font-medium transition ${filter === ''
                  ? 'bg-violet-500/20 text-violet-100 border border-violet-400/40'
                  : 'text-slate-400 hover:text-white'
                  }`}
                onClick={() => handleFilterChange('')}
              >
                All
              </button>
              <button
                type="button"
                className={`rounded-lg px-3 py-1.5 text-xs font-medium transition flex items-center gap-1.5 ${filter === 'openai'
                  ? 'bg-emerald-500/20 text-emerald-100 border border-emerald-400/40'
                  : 'text-slate-400 hover:text-white'
                  }`}
                onClick={() => handleFilterChange('openai')}
              >
                <Terminal size={12} />
                OpenAI
              </button>
              <button
                type="button"
                className={`rounded-lg px-3 py-1.5 text-xs font-medium transition flex items-center gap-1.5 ${filter === 'anthropic'
                  ? 'bg-amber-500/20 text-amber-100 border border-amber-400/40'
                  : 'text-slate-400 hover:text-white'
                  }`}
                onClick={() => handleFilterChange('anthropic')}
              >
                <MessageSquare size={12} />
                Anthropic
              </button>
              <button
                type="button"
                className={`rounded-lg px-3 py-1.5 text-xs font-medium transition flex items-center gap-1.5 ${filter === 'responses'
                  ? 'bg-rose-500/20 text-rose-100 border border-rose-400/40'
                  : 'text-slate-400 hover:text-white'
                  }`}
                onClick={() => handleFilterChange('responses')}
              >
                <Bot size={12} />
                Responses
              </button>
            </div>
            <button
              type="button"
              onClick={handleRefresh}
              className="inline-flex items-center gap-2 rounded-xl border border-white/15 bg-slate-800/50 px-4 py-2 text-sm text-slate-200 transition-colors hover:border-white/30 hover:text-white"
            >
              <RefreshCcw size={16} />
              Refresh
            </button>
          </div>
        }
      />

      {interactionsError && (
        <div className="rounded-xl border border-amber-400/40 bg-amber-500/10 p-4 text-amber-100">
          {interactionsError}
        </div>
      )}

      {/* Stats bar - show counts by frontdoor type */}
      <div className="flex flex-wrap gap-3">
        <div className="rounded-xl border border-white/10 bg-slate-900/60 px-4 py-2 flex items-center gap-3">
          <Terminal size={16} className="text-emerald-300" />
          <span className="text-sm text-white font-medium">{openaiCount}</span>
          <span className="text-xs text-slate-400">OpenAI</span>
        </div>
        <div className="rounded-xl border border-white/10 bg-slate-900/60 px-4 py-2 flex items-center gap-3">
          <MessageSquare size={16} className="text-amber-300" />
          <span className="text-sm text-white font-medium">{anthropicCount}</span>
          <span className="text-xs text-slate-400">Anthropic</span>
        </div>
        <div className="rounded-xl border border-white/10 bg-slate-900/60 px-4 py-2 flex items-center gap-3">
          <Bot size={16} className="text-rose-300" />
          <span className="text-sm text-white font-medium">{responsesCount}</span>
          <span className="text-xs text-slate-400">Responses</span>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-5 lg:grid-cols-[420px_1fr]">
        {/* Interaction List */}
        <div className="rounded-2xl border border-white/10 bg-slate-900/70">
          <div className="flex items-center justify-between border-b border-white/10 px-4 py-3">
            <div className="flex items-center gap-2 text-sm font-semibold text-white">
              <Database size={16} className="text-violet-300" />
              Interactions
            </div>
            <Pill icon={Database} label={`${interactions.length} `} />
          </div>

          <div className="max-h-[calc(100vh-400px)] min-h-[400px] space-y-2 overflow-y-auto p-3">
            {loadingInteractions && <LoadingState message="Loading interactions..." />}

            {!loadingInteractions && interactions.length === 0 && (
              <EmptyState
                icon={Database}
                title="No interactions yet"
                description="Interactions will appear here as they are recorded"
              />
            )}

            {!loadingInteractions &&
              interactions.map((interaction) => {
                // Determine frontdoor type for icon/color
                const frontdoor = getFrontdoor(interaction.metadata);
                const isOpenAI = frontdoor === 'openai';
                const isAnthropic = frontdoor === 'anthropic';
                const isResponses = frontdoor === 'responses';
                const metadata = interaction.metadata as Record<string, string> | null;

                return (
                  <button
                    key={interaction.id}
                    onClick={() => openInteraction(interaction.id)}
                    className={`group flex w-full flex-col gap-2 rounded-xl border px-4 py-3 text-left transition ${selectedId === interaction.id
                      ? 'border-violet-400/50 bg-violet-500/10 shadow-[0_0_20px_rgba(139,92,246,0.1)]'
                      : 'border-white/10 bg-white/5 hover:border-white/20 hover:bg-white/10'
                      }`}
                  >
                    <div className="flex items-center justify-between gap-2">
                      <div className="flex items-center gap-2 min-w-0">
                        {isOpenAI ? (
                          <Terminal size={14} className="text-emerald-300 shrink-0" />
                        ) : isAnthropic ? (
                          <MessageSquare size={14} className="text-amber-300 shrink-0" />
                        ) : isResponses ? (
                          <Bot size={14} className="text-rose-300 shrink-0" />
                        ) : (
                          <ArrowLeftRight size={14} className="text-violet-300 shrink-0" />
                        )}
                        <span className="truncate text-sm font-semibold text-white">
                          {interaction.id.slice(0, 20)}
                        </span>
                      </div>
                      <div className="flex items-center gap-2 shrink-0">
                        {interaction.status && (
                          <StatusBadge status={interaction.status} />
                        )}
                      </div>
                    </div>

                    <div className="flex items-center gap-2 text-[11px] text-slate-400">
                      <Clock4 size={12} />
                      <span>{formatShortDate(interaction.updatedAt)}</span>
                      {interaction.model && (
                        <>
                          <span className="text-slate-600">•</span>
                          <span className="text-slate-300">{interaction.model}</span>
                        </>
                      )}
                    </div>

                    <div className="flex flex-wrap gap-1.5">
                      {/* Frontdoor badge */}
                      <span className={`rounded-md px-2 py-0.5 text-[10px] font-medium ${isOpenAI
                        ? 'bg-emerald-500/20 text-emerald-200'
                        : isAnthropic
                          ? 'bg-amber-500/20 text-amber-200'
                          : isResponses
                            ? 'bg-rose-500/20 text-rose-200'
                            : 'bg-violet-500/20 text-violet-200'
                        }`}>
                        {frontdoor}
                      </span>
                      {metadata?.provider && (
                        <span className="rounded-md bg-slate-800/80 px-2 py-0.5 text-[10px] text-emerald-200">
                          prov: {metadata.provider}
                        </span>
                      )}
                      {metadata?.app && (
                        <span className="rounded-md bg-slate-800/80 px-2 py-0.5 text-[10px] text-slate-200">
                          app: {metadata.app}
                        </span>
                      )}
                      {interaction.previousResponseId && (
                        <span className="rounded-md bg-slate-800/80 px-2 py-0.5 text-[10px] text-emerald-200">
                          <ArrowLeftRight size={10} className="inline mr-1" />
                          continues
                        </span>
                      )}
                      {interaction.status === 'incomplete' && (
                        <span className="rounded-md bg-violet-500/20 px-2 py-0.5 text-[10px] text-violet-200">
                          <Wrench size={10} className="inline mr-1" />
                          tool use
                        </span>
                      )}
                    </div>
                  </button>
                );
              })}
          </div>
        </div>

        {/* Interaction Detail */}
        <div className="rounded-2xl border border-white/10 bg-slate-900/70">
          {!selectedInteraction && !loadingDetail && (
            <div className="flex h-full min-h-[400px] flex-col items-center justify-center gap-4 text-slate-400">
              <Database size={48} className="text-slate-600" />
              <div className="text-center">
                <div className="text-sm font-medium">Select an interaction</div>
                <div className="text-xs text-slate-500 mt-1">
                  Click on an interaction to view details
                </div>
              </div>
            </div>
          )}

          {loadingDetail && (
            <div className="flex h-full min-h-[400px] flex-col items-center justify-center gap-3 text-slate-300">
              <Loader2 className="h-8 w-8 animate-spin text-violet-400" />
              <div className="text-sm">Loading details...</div>
            </div>
          )}

          {selectedInteraction && !loadingDetail && (
            <UnifiedInteractionDetail
              interaction={selectedInteraction}
            />
          )}
        </div>
      </div>
    </div>
  );
}