import { useState } from 'react';
import {
  ArrowLeftRight,
  Bot,
  CheckCircle,
  Clock4,
  Loader2,
  RefreshCcw,
  ServerCog,
  Shield,
  Zap,
} from 'lucide-react';
import { useApi, formatShortDate } from '../hooks/useApi';
import { PageHeader, Pill, EmptyState, LoadingState, StatusBadge } from '../components/ui';
import type { ResponseDetail } from '../types';

export function Responses() {
  const { overview, responses, loadingResponses, responsesError, refreshResponses, fetchResponseDetail } = useApi();
  const [selectedResponse, setSelectedResponse] = useState<ResponseDetail | null>(null);
  const [loadingResponseDetail, setLoadingResponseDetail] = useState(false);

  const openResponse = async (responseId: string) => {
    setLoadingResponseDetail(true);
    const detail = await fetchResponseDetail(responseId);
    setSelectedResponse(detail);
    setLoadingResponseDetail(false);
  };

  const handleRefresh = () => {
    setSelectedResponse(null);
    refreshResponses();
  };

  // Check if any app has responses API enabled
  const hasResponsesEnabled = (overview?.apps ?? []).some((app) => app?.enable_responses);

  if (!overview?.storage.enabled) {
    return (
      <div className="space-y-6">
        <PageHeader
          title="Responses API"
          subtitle="OpenAI Responses API records"
          icon={Bot}
          iconColor="text-rose-300"
        />
        <div className="rounded-2xl border border-amber-400/30 bg-amber-500/10 p-8 text-center">
          <Zap size={48} className="mx-auto mb-4 text-amber-300" />
          <h2 className="text-xl font-semibold text-white mb-2">Storage Disabled</h2>
          <p className="text-slate-400 max-w-md mx-auto">
            Response storage is disabled. Enable storage in your gateway configuration to view Responses API records.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Responses API"
        subtitle={`${responses.length} response record${responses.length !== 1 ? 's' : ''} stored`}
        icon={Bot}
        iconColor="text-rose-300"
        actions={
          <div className="flex items-center gap-3">
            {hasResponsesEnabled && (
              <Pill icon={CheckCircle} label="API enabled" tone="emerald" />
            )}
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

      {responsesError && (
        <div className="rounded-xl border border-amber-400/40 bg-amber-500/10 p-4 text-amber-100">
          {responsesError}
        </div>
      )}

      {!hasResponsesEnabled && (
        <div className="rounded-2xl border border-slate-700/50 bg-slate-800/30 p-5">
          <div className="flex items-start gap-4">
            <div className="rounded-xl bg-slate-700/50 p-3 text-slate-400">
              <Bot size={24} />
            </div>
            <div>
              <h3 className="text-sm font-semibold text-white">Responses API not enabled</h3>
              <p className="text-xs text-slate-400 mt-1">
                Enable the Responses API on one or more apps in your gateway configuration to start recording responses.
                Add <code className="text-amber-200/70">enable_responses: true</code> to your app configuration.
              </p>
            </div>
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 gap-5 lg:grid-cols-[400px_1fr]">
        {/* Response List */}
        <div className="rounded-2xl border border-white/10 bg-slate-900/70">
          <div className="flex items-center justify-between border-b border-white/10 px-4 py-3">
            <div className="flex items-center gap-2 text-sm font-semibold text-white">
              <Bot size={16} className="text-rose-300" />
              Responses
            </div>
            <Pill icon={Bot} label={`${responses.length}`} />
          </div>

          <div className="max-h-[calc(100vh-380px)] min-h-[400px] space-y-2 overflow-y-auto p-3">
            {loadingResponses && <LoadingState message="Loading responses..." />}

            {!loadingResponses && responses.length === 0 && (
              <EmptyState
                icon={Bot}
                title="No responses yet"
                description="Use the Responses API endpoint to create records"
              />
            )}

            {!loadingResponses &&
              responses.map((response) => (
                <button
                  key={response.id}
                  onClick={() => openResponse(response.id)}
                  className={`group flex w-full flex-col gap-2 rounded-xl border px-4 py-3 text-left transition ${
                    selectedResponse?.id === response.id
                      ? 'border-rose-400/50 bg-rose-500/10 shadow-[0_0_20px_rgba(244,63,94,0.1)]'
                      : 'border-white/10 bg-white/5 hover:border-white/20 hover:bg-white/10'
                  }`}
                >
                  <div className="flex items-center justify-between">
                    <div className="truncate text-sm font-semibold text-white font-mono">
                      {response.id.slice(0, 20)}...
                    </div>
                    <StatusBadge status={response.status} />
                  </div>

                  <div className="flex items-center gap-2 text-[11px] text-slate-400">
                    <Clock4 size={12} />
                    <span>{formatShortDate(response.updated_at)}</span>
                  </div>

                  <div className="flex flex-wrap gap-1.5">
                    <span className="rounded-md bg-slate-800/80 px-2 py-0.5 text-[10px] text-slate-200">
                      {response.model}
                    </span>
                    {response.previous_response_id && (
                      <span className="rounded-md bg-slate-800/80 px-2 py-0.5 text-[10px] text-emerald-200">
                        <ArrowLeftRight size={10} className="inline mr-1" />
                        continuation
                      </span>
                    )}
                  </div>
                </button>
              ))}
          </div>
        </div>

        {/* Response Detail */}
        <div className="rounded-2xl border border-white/10 bg-slate-900/70">
          {!selectedResponse && !loadingResponseDetail && (
            <div className="flex h-full min-h-[400px] flex-col items-center justify-center gap-4 text-slate-400">
              <Bot size={48} className="text-slate-600" />
              <div className="text-center">
                <div className="text-sm font-medium">Select a response</div>
                <div className="text-xs text-slate-500 mt-1">
                  Click on a response to view details
                </div>
              </div>
            </div>
          )}

          {loadingResponseDetail && (
            <div className="flex h-full min-h-[400px] flex-col items-center justify-center gap-3 text-slate-300">
              <Loader2 className="h-8 w-8 animate-spin text-rose-400" />
              <div className="text-sm">Loading response...</div>
            </div>
          )}

          {selectedResponse && !loadingResponseDetail && (
            <div className="flex h-full flex-col">
              {/* Response Header */}
              <div className="border-b border-white/10 px-5 py-4">
                <div className="flex flex-col gap-3">
                  <div className="flex items-center justify-between">
                    <div className="text-sm font-mono font-semibold text-white break-all">
                      {selectedResponse.id}
                    </div>
                    <StatusBadge status={selectedResponse.status} />
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <span className="inline-flex items-center gap-1.5 rounded-md bg-slate-800/80 px-2.5 py-1 text-xs text-slate-300">
                      <Clock4 size={12} />
                      Created: {formatShortDate(selectedResponse.created_at)}
                    </span>
                    <span className="inline-flex items-center gap-1.5 rounded-md bg-slate-800/80 px-2.5 py-1 text-xs text-slate-300">
                      <ServerCog size={12} />
                      Model: {selectedResponse.model}
                    </span>
                    {selectedResponse.previous_response_id && (
                      <span className="inline-flex items-center gap-1.5 rounded-md bg-emerald-500/20 px-2.5 py-1 text-xs text-emerald-200">
                        <ArrowLeftRight size={12} />
                        Continues: {selectedResponse.previous_response_id.slice(0, 12)}...
                      </span>
                    )}
                  </div>
                </div>
              </div>

              {/* Response Content */}
              <div className="flex-1 space-y-4 overflow-y-auto px-5 py-4 max-h-[calc(100vh-480px)]">
                {/* Request */}
                {selectedResponse.request && (
                  <div className="rounded-2xl border border-amber-500/20 bg-amber-500/5 px-4 py-3">
                    <div className="flex items-center gap-2 text-xs text-slate-400 mb-3">
                      <Shield size={14} className="text-amber-300" />
                      <span className="font-semibold text-amber-200">Request</span>
                    </div>
                    <pre className="whitespace-pre-wrap text-xs text-slate-300 bg-slate-950/50 p-4 rounded-xl overflow-auto max-h-[250px] font-mono">
                      {JSON.stringify(selectedResponse.request, null, 2)}
                    </pre>
                  </div>
                )}

                {/* Response */}
                {selectedResponse.response && (
                  <div className="rounded-2xl border border-emerald-500/20 bg-emerald-500/5 px-4 py-3">
                    <div className="flex items-center gap-2 text-xs text-slate-400 mb-3">
                      <Bot size={14} className="text-emerald-300" />
                      <span className="font-semibold text-emerald-200">Response</span>
                    </div>
                    <pre className="whitespace-pre-wrap text-xs text-slate-300 bg-slate-950/50 p-4 rounded-xl overflow-auto max-h-[350px] font-mono">
                      {JSON.stringify(selectedResponse.response, null, 2)}
                    </pre>
                  </div>
                )}

                {/* Metadata */}
                {selectedResponse.metadata && Object.keys(selectedResponse.metadata).length > 0 && (
                  <div className="rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3">
                    <div className="flex items-center gap-2 text-xs text-slate-400 mb-3">
                      <ServerCog size={14} className="text-slate-400" />
                      <span className="font-semibold text-slate-300">Metadata</span>
                    </div>
                    <div className="space-y-1">
                      {Object.entries(selectedResponse.metadata).map(([key, value]) => (
                        <div key={key} className="flex items-center gap-2 text-xs">
                          <span className="text-slate-500">{key}:</span>
                          <span className="text-white">{value}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {!selectedResponse.request && !selectedResponse.response && (
                  <div className="text-sm text-slate-500 text-center py-8">
                    No request/response data available.
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Status Summary */}
      {responses.length > 0 && (
        <div className="rounded-2xl border border-white/10 bg-slate-900/60 p-5">
          <h2 className="text-sm font-semibold text-white mb-4">Response Status Summary</h2>
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
            <div className="rounded-xl border border-emerald-500/20 bg-emerald-500/5 p-4 text-center">
              <div className="text-2xl font-bold text-emerald-200">
                {responses.filter((r) => r.status === 'completed').length}
              </div>
              <div className="text-xs text-slate-400 mt-1">Completed</div>
            </div>
            <div className="rounded-xl border border-amber-500/20 bg-amber-500/5 p-4 text-center">
              <div className="text-2xl font-bold text-amber-200">
                {responses.filter((r) => r.status === 'in_progress').length}
              </div>
              <div className="text-xs text-slate-400 mt-1">In Progress</div>
            </div>
            <div className="rounded-xl border border-red-500/20 bg-red-500/5 p-4 text-center">
              <div className="text-2xl font-bold text-red-200">
                {responses.filter((r) => r.status === 'failed').length}
              </div>
              <div className="text-xs text-slate-400 mt-1">Failed</div>
            </div>
            <div className="rounded-xl border border-slate-500/20 bg-slate-500/5 p-4 text-center">
              <div className="text-2xl font-bold text-slate-200">
                {responses.filter((r) => r.status === 'cancelled').length}
              </div>
              <div className="text-xs text-slate-400 mt-1">Cancelled</div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
