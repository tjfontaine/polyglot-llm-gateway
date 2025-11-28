import { useState } from 'react';
import {
  ArrowLeftRight,
  Bot,
  Clock4,
  Compass,
  Database,
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
import { useApi, formatShortDate } from '../hooks/useApi';
import { PageHeader, Pill, EmptyState, LoadingState, StatusBadge } from '../components/ui';
import type { InteractionDetail, ResponseOutputItem, ResponseData } from '../types';

type FilterType = '' | 'conversation' | 'response';

// Helper component to render a single output item
function OutputItemCard({ item }: { item: ResponseOutputItem }) {
  if (item.type === 'message') {
    const text = item.content
      ?.filter(part => part.type === 'output_text')
      .map(part => part.text)
      .join('') || '';
    
    return (
      <div className="rounded-xl border border-emerald-500/20 bg-emerald-500/5 px-4 py-3">
        <div className="flex items-center gap-2 text-xs text-slate-400 mb-2">
          <Bot size={14} className="text-emerald-300" />
          <span className="font-semibold text-emerald-200">Assistant Message</span>
          {item.status && <StatusBadge status={item.status} />}
        </div>
        <p className="whitespace-pre-wrap text-sm leading-relaxed text-white">
          {text || <span className="text-slate-500 italic">No content</span>}
        </p>
      </div>
    );
  }

  if (item.type === 'function_call') {
    // Try to format arguments as pretty JSON
    let formattedArgs = item.arguments || '{}';
    try {
      formattedArgs = JSON.stringify(JSON.parse(item.arguments || '{}'), null, 2);
    } catch {
      // Keep original if not valid JSON
    }

    return (
      <div className="rounded-xl border border-violet-500/30 bg-violet-500/10 px-4 py-3">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <div className="p-1.5 rounded-lg bg-violet-500/20">
              <Wrench size={14} className="text-violet-300" />
            </div>
            <div>
              <span className="font-semibold text-violet-200 text-sm">Tool Call</span>
              <span className="text-slate-400 text-xs ml-2">→ {item.name}</span>
            </div>
          </div>
          {item.status && <StatusBadge status={item.status} />}
        </div>
        
        <div className="space-y-2">
          <div className="flex items-center gap-2">
            <span className="text-xs text-slate-500">Function:</span>
            <code className="px-2 py-0.5 rounded bg-slate-800/80 text-xs text-violet-200 font-mono">
              {item.name}
            </code>
          </div>
          {item.call_id && (
            <div className="flex items-center gap-2">
              <span className="text-xs text-slate-500">Call ID:</span>
              <code className="px-2 py-0.5 rounded bg-slate-800/80 text-xs text-slate-300 font-mono">
                {item.call_id}
              </code>
            </div>
          )}
          <div>
            <div className="flex items-center gap-2 mb-1.5">
              <Terminal size={12} className="text-slate-400" />
              <span className="text-xs text-slate-400">Arguments</span>
            </div>
            <pre className="whitespace-pre-wrap text-xs text-slate-300 bg-slate-950/50 p-3 rounded-lg overflow-auto max-h-[200px] font-mono border border-white/5">
              {formattedArgs}
            </pre>
          </div>
        </div>
      </div>
    );
  }

  if (item.type === 'function_call_output') {
    return (
      <div className="rounded-xl border border-cyan-500/30 bg-cyan-500/10 px-4 py-3">
        <div className="flex items-center gap-2 text-xs text-slate-400 mb-2">
          <Terminal size={14} className="text-cyan-300" />
          <span className="font-semibold text-cyan-200">Tool Result</span>
          {item.call_id && (
            <code className="px-1.5 py-0.5 rounded bg-slate-800/80 text-[10px] text-slate-300 font-mono">
              {item.call_id}
            </code>
          )}
        </div>
        <pre className="whitespace-pre-wrap text-xs text-slate-300 bg-slate-950/50 p-3 rounded-lg overflow-auto max-h-[150px] font-mono">
          {item.output || 'No output'}
        </pre>
      </div>
    );
  }

  // Fallback for unknown types
  return (
    <div className="rounded-xl border border-slate-500/30 bg-slate-500/10 px-4 py-3">
      <div className="flex items-center gap-2 text-xs text-slate-400 mb-2">
        <span className="font-semibold">Unknown: {item.type}</span>
      </div>
      <pre className="whitespace-pre-wrap text-xs text-slate-300 bg-slate-950/50 p-3 rounded-lg overflow-auto max-h-[150px] font-mono">
        {JSON.stringify(item, null, 2)}
      </pre>
    </div>
  );
}

// Helper to render the response section with improved tool call display
function ResponseSection({ response }: { response: ResponseData }) {
  const hasOutput = response.output && response.output.length > 0;
  const hasToolCalls = response.output?.some(item => item.type === 'function_call') || false;

  return (
    <div className="space-y-4">
      {/* Status bar for tool calls */}
      {hasToolCalls && (
        <div className="flex items-center gap-3 px-4 py-3 rounded-xl bg-violet-500/10 border border-violet-500/20">
          <Wrench size={16} className="text-violet-300" />
          <span className="text-sm text-violet-200 font-medium">
            This response includes tool calls
          </span>
          <span className="px-2 py-0.5 rounded-full bg-violet-500/20 text-xs text-violet-200">
            {response.output?.filter(i => i.type === 'function_call').length} tool{response.output?.filter(i => i.type === 'function_call').length !== 1 ? 's' : ''}
          </span>
        </div>
      )}

      {/* Output items */}
      {hasOutput ? (
        <div className="space-y-3">
          <div className="text-xs text-slate-400 font-medium uppercase tracking-wide px-1">
            Output ({response.output?.length} item{response.output?.length !== 1 ? 's' : ''})
          </div>
          {response.output?.map((item, idx) => (
            <OutputItemCard key={item.id || idx} item={item} />
          ))}
        </div>
      ) : (
        <div className="rounded-2xl border border-emerald-500/20 bg-emerald-500/5 px-4 py-3">
          <div className="flex items-center gap-2 text-xs text-slate-400 mb-3">
            <Bot size={14} className="text-emerald-300" />
            <span className="font-semibold text-emerald-200">Response</span>
          </div>
          <pre className="whitespace-pre-wrap text-xs text-slate-300 bg-slate-950/50 p-4 rounded-xl overflow-auto max-h-[350px] font-mono">
            {JSON.stringify(response, null, 2)}
          </pre>
        </div>
      )}

      {/* Usage info */}
      {response.usage && (
        <div className="flex flex-wrap gap-3 text-xs">
          {response.usage.input_tokens !== undefined && (
            <span className="px-2.5 py-1 rounded-lg bg-slate-800/60 text-slate-300">
              Input: <span className="text-white font-medium">{response.usage.input_tokens}</span> tokens
            </span>
          )}
          {response.usage.output_tokens !== undefined && (
            <span className="px-2.5 py-1 rounded-lg bg-slate-800/60 text-slate-300">
              Output: <span className="text-white font-medium">{response.usage.output_tokens}</span> tokens
            </span>
          )}
          {response.usage.total_tokens !== undefined && (
            <span className="px-2.5 py-1 rounded-lg bg-slate-800/60 text-slate-300">
              Total: <span className="text-white font-medium">{response.usage.total_tokens}</span> tokens
            </span>
          )}
        </div>
      )}
    </div>
  );
}

export function Data() {
  const { overview, interactions, interactionsTotal, loadingInteractions, interactionsError, refreshInteractions, fetchInteractionDetail } = useApi();
  const [selectedInteraction, setSelectedInteraction] = useState<InteractionDetail | null>(null);
  const [loadingDetail, setLoadingDetail] = useState(false);
  const [filter, setFilter] = useState<FilterType>('');

  const openInteraction = async (id: string) => {
    setLoadingDetail(true);
    const detail = await fetchInteractionDetail(id);
    setSelectedInteraction(detail);
    setLoadingDetail(false);
  };

  const handleRefresh = () => {
    setSelectedInteraction(null);
    refreshInteractions(filter);
  };

  const handleFilterChange = (newFilter: FilterType) => {
    setFilter(newFilter);
    setSelectedInteraction(null);
    refreshInteractions(newFilter);
  };

  if (!overview?.storage.enabled) {
    return (
      <div className="space-y-6">
        <PageHeader
          title="Data Explorer"
          subtitle="Conversations & API responses"
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

  const conversationCount = interactions.filter(i => i.type === 'conversation').length;
  const responseCount = interactions.filter(i => i.type === 'response').length;

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
                className={`rounded-lg px-3 py-1.5 text-xs font-medium transition ${
                  filter === ''
                    ? 'bg-violet-500/20 text-violet-100 border border-violet-400/40'
                    : 'text-slate-400 hover:text-white'
                }`}
                onClick={() => handleFilterChange('')}
              >
                All
              </button>
              <button
                type="button"
                className={`rounded-lg px-3 py-1.5 text-xs font-medium transition flex items-center gap-1.5 ${
                  filter === 'conversation'
                    ? 'bg-sky-500/20 text-sky-100 border border-sky-400/40'
                    : 'text-slate-400 hover:text-white'
                }`}
                onClick={() => handleFilterChange('conversation')}
              >
                <MessageSquare size={12} />
                Chats
              </button>
              <button
                type="button"
                className={`rounded-lg px-3 py-1.5 text-xs font-medium transition flex items-center gap-1.5 ${
                  filter === 'response'
                    ? 'bg-rose-500/20 text-rose-100 border border-rose-400/40'
                    : 'text-slate-400 hover:text-white'
                }`}
                onClick={() => handleFilterChange('response')}
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

      {/* Stats bar */}
      <div className="flex flex-wrap gap-3">
        <div className="rounded-xl border border-white/10 bg-slate-900/60 px-4 py-2 flex items-center gap-3">
          <MessageSquare size={16} className="text-sky-300" />
          <span className="text-sm text-white font-medium">{conversationCount}</span>
          <span className="text-xs text-slate-400">conversations</span>
        </div>
        <div className="rounded-xl border border-white/10 bg-slate-900/60 px-4 py-2 flex items-center gap-3">
          <Bot size={16} className="text-rose-300" />
          <span className="text-sm text-white font-medium">{responseCount}</span>
          <span className="text-xs text-slate-400">responses</span>
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
            <Pill icon={Database} label={`${interactions.length}`} />
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
              interactions.map((interaction) => (
                <button
                  key={interaction.id}
                  onClick={() => openInteraction(interaction.id)}
                  className={`group flex w-full flex-col gap-2 rounded-xl border px-4 py-3 text-left transition ${
                    selectedInteraction?.id === interaction.id
                      ? 'border-violet-400/50 bg-violet-500/10 shadow-[0_0_20px_rgba(139,92,246,0.1)]'
                      : 'border-white/10 bg-white/5 hover:border-white/20 hover:bg-white/10'
                  }`}
                >
                  <div className="flex items-center justify-between gap-2">
                    <div className="flex items-center gap-2 min-w-0">
                      {interaction.type === 'conversation' ? (
                        <MessageSquare size={14} className="text-sky-300 flex-shrink-0" />
                      ) : (
                        <Bot size={14} className="text-rose-300 flex-shrink-0" />
                      )}
                      <span className="truncate text-sm font-semibold text-white">
                        {interaction.type === 'conversation'
                          ? interaction.metadata?.title || interaction.id.slice(0, 16)
                          : interaction.id.slice(0, 20)}
                      </span>
                    </div>
                    <div className="flex items-center gap-2 flex-shrink-0">
                      {interaction.type === 'conversation' && interaction.message_count && (
                        <span className="rounded-full bg-sky-500/20 px-2 py-0.5 text-xs font-medium text-sky-200">
                          {interaction.message_count} msg
                        </span>
                      )}
                      {interaction.type === 'response' && interaction.status && (
                        <StatusBadge status={interaction.status} />
                      )}
                    </div>
                  </div>

                  <div className="flex items-center gap-2 text-[11px] text-slate-400">
                    <Clock4 size={12} />
                    <span>{formatShortDate(interaction.updated_at)}</span>
                    {interaction.model && (
                      <>
                        <span className="text-slate-600">•</span>
                        <span className="text-slate-300">{interaction.model}</span>
                      </>
                    )}
                  </div>

                  <div className="flex flex-wrap gap-1.5">
                    <span className={`rounded-md px-2 py-0.5 text-[10px] font-medium ${
                      interaction.type === 'conversation'
                        ? 'bg-sky-500/20 text-sky-200'
                        : 'bg-rose-500/20 text-rose-200'
                    }`}>
                      {interaction.type}
                    </span>
                    {interaction.metadata?.frontdoor && (
                      <span className="rounded-md bg-slate-800/80 px-2 py-0.5 text-[10px] text-slate-300">
                        fd: {interaction.metadata.frontdoor}
                      </span>
                    )}
                    {interaction.metadata?.provider && (
                      <span className="rounded-md bg-slate-800/80 px-2 py-0.5 text-[10px] text-emerald-200">
                        prov: {interaction.metadata.provider}
                      </span>
                    )}
                    {interaction.metadata?.app && (
                      <span className="rounded-md bg-slate-800/80 px-2 py-0.5 text-[10px] text-slate-200">
                        app: {interaction.metadata.app}
                      </span>
                    )}
                    {interaction.previous_response_id && (
                      <span className="rounded-md bg-slate-800/80 px-2 py-0.5 text-[10px] text-emerald-200">
                        <ArrowLeftRight size={10} className="inline mr-1" />
                        continues
                      </span>
                    )}
                    {interaction.type === 'response' && interaction.status === 'incomplete' && (
                      <span className="rounded-md bg-violet-500/20 px-2 py-0.5 text-[10px] text-violet-200">
                        <Wrench size={10} className="inline mr-1" />
                        tool use
                      </span>
                    )}
                  </div>
                </button>
              ))}
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
            <div className="flex h-full flex-col">
              {/* Detail Header */}
              <div className="border-b border-white/10 px-5 py-4">
                <div className="flex flex-col gap-3">
                  <div className="flex items-center gap-3">
                    {selectedInteraction.type === 'conversation' ? (
                      <div className="rounded-lg bg-sky-500/10 p-2 text-sky-300">
                        <MessageSquare size={20} />
                      </div>
                    ) : (
                      <div className="rounded-lg bg-rose-500/10 p-2 text-rose-300">
                        <Bot size={20} />
                      </div>
                    )}
                    <div className="min-w-0 flex-1">
                      <div className="text-sm font-mono font-semibold text-white truncate">
                        {selectedInteraction.type === 'conversation'
                          ? selectedInteraction.metadata?.title || selectedInteraction.id
                          : selectedInteraction.id}
                      </div>
                      <div className="flex items-center gap-2 mt-1">
                        <span className={`rounded-md px-2 py-0.5 text-xs font-medium ${
                          selectedInteraction.type === 'conversation'
                            ? 'bg-sky-500/20 text-sky-200'
                            : 'bg-rose-500/20 text-rose-200'
                        }`}>
                          {selectedInteraction.type}
                        </span>
                        {selectedInteraction.status && (
                          <StatusBadge status={selectedInteraction.status} />
                        )}
                      </div>
                    </div>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <span className="inline-flex items-center gap-1.5 rounded-md bg-slate-800/80 px-2.5 py-1 text-xs text-slate-300">
                      <Clock4 size={12} />
                      {formatShortDate(selectedInteraction.created_at)}
                    </span>
                    {selectedInteraction.model && (
                      <span className="inline-flex items-center gap-1.5 rounded-md bg-slate-800/80 px-2.5 py-1 text-xs text-slate-300">
                        <ServerCog size={12} />
                        {selectedInteraction.model}
                      </span>
                    )}
                    {selectedInteraction.metadata?.frontdoor && (
                      <span className="inline-flex items-center gap-1.5 rounded-md bg-slate-800/80 px-2.5 py-1 text-xs text-slate-300">
                        <Route size={12} />
                        fd: {selectedInteraction.metadata.frontdoor}
                      </span>
                    )}
                    {selectedInteraction.metadata?.provider && (
                      <span className="inline-flex items-center gap-1.5 rounded-md bg-slate-800/80 px-2.5 py-1 text-xs text-emerald-200">
                        <ServerCog size={12} />
                        provider: {selectedInteraction.metadata.provider}
                      </span>
                    )}
                    {selectedInteraction.metadata?.app && (
                      <span className="inline-flex items-center gap-1.5 rounded-md bg-slate-800/80 px-2.5 py-1 text-xs text-slate-200">
                        <Compass size={12} />
                        app: {selectedInteraction.metadata.app}
                      </span>
                    )}
                    {selectedInteraction.previous_response_id && (
                      <span className="inline-flex items-center gap-1.5 rounded-md bg-emerald-500/20 px-2.5 py-1 text-xs text-emerald-200">
                        <ArrowLeftRight size={12} />
                        continues: {selectedInteraction.previous_response_id.slice(0, 12)}...
                      </span>
                    )}
                  </div>
                </div>
              </div>

              {/* Detail Content */}
              <div className="flex-1 space-y-4 overflow-y-auto px-5 py-4 max-h-[calc(100vh-500px)]">
                {/* Conversation Messages */}
                {selectedInteraction.type === 'conversation' && selectedInteraction.messages && (
                  <>
                    {selectedInteraction.messages.map((msg) => (
                      <div
                        key={msg.id}
                        className={`rounded-2xl border px-4 py-3 ${
                          msg.role === 'user'
                            ? 'border-amber-500/20 bg-amber-500/5'
                            : 'border-emerald-500/20 bg-emerald-500/5'
                        }`}
                      >
                        <div className="flex items-center justify-between text-xs text-slate-400 mb-2">
                          <span className="inline-flex items-center gap-1.5 font-medium">
                            {msg.role === 'user' ? (
                              <>
                                <Shield size={14} className="text-amber-300" />
                                <span className="text-amber-200">User</span>
                              </>
                            ) : (
                              <>
                                <ServerCog size={14} className="text-emerald-300" />
                                <span className="text-emerald-200">Assistant</span>
                              </>
                            )}
                          </span>
                          <span>{formatShortDate(msg.created_at)}</span>
                        </div>
                        <p className="whitespace-pre-wrap text-sm leading-relaxed text-white">
                          {msg.content}
                        </p>
                      </div>
                    ))}
                    {selectedInteraction.messages.length === 0 && (
                      <div className="text-sm text-slate-500 text-center py-8">
                        No messages in this conversation.
                      </div>
                    )}
                  </>
                )}

                {/* Response Request/Response */}
                {selectedInteraction.type === 'response' && (
                  <>
                    {selectedInteraction.request && (
                      <div className="rounded-2xl border border-amber-500/20 bg-amber-500/5 px-4 py-3">
                        <div className="flex items-center gap-2 text-xs text-slate-400 mb-3">
                          <Shield size={14} className="text-amber-300" />
                          <span className="font-semibold text-amber-200">Request</span>
                        </div>
                        <pre className="whitespace-pre-wrap text-xs text-slate-300 bg-slate-950/50 p-4 rounded-xl overflow-auto max-h-[250px] font-mono">
                          {JSON.stringify(selectedInteraction.request, null, 2)}
                        </pre>
                      </div>
                    )}

                    {selectedInteraction.response && (
                      <ResponseSection response={selectedInteraction.response} />
                    )}

                    {!selectedInteraction.request && !selectedInteraction.response && (
                      <div className="text-sm text-slate-500 text-center py-8">
                        No request/response data available.
                      </div>
                    )}
                  </>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
