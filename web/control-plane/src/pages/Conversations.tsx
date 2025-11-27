import { useState } from 'react';
import {
  Clock4,
  Compass,
  Loader2,
  MessageSquare,
  RefreshCcw,
  Route,
  ServerCog,
  Shield,
  Signal,
  Zap,
} from 'lucide-react';
import { useApi, formatShortDate } from '../hooks/useApi';
import { PageHeader, Pill, EmptyState, LoadingState } from '../components/ui';
import type { ThreadDetail } from '../types';

export function Conversations() {
  const { overview, threads, loadingThreads, threadsError, refreshThreads, fetchThreadDetail } = useApi();
  const [selectedThread, setSelectedThread] = useState<ThreadDetail | null>(null);
  const [loadingThreadDetail, setLoadingThreadDetail] = useState(false);

  const openThread = async (threadId: string) => {
    setLoadingThreadDetail(true);
    const detail = await fetchThreadDetail(threadId);
    setSelectedThread(detail);
    setLoadingThreadDetail(false);
  };

  const handleRefresh = () => {
    setSelectedThread(null);
    refreshThreads();
  };

  if (!overview?.storage.enabled) {
    return (
      <div className="space-y-6">
        <PageHeader
          title="Conversations"
          subtitle="Chat threads and messages"
          icon={MessageSquare}
          iconColor="text-sky-300"
        />
        <div className="rounded-2xl border border-amber-400/30 bg-amber-500/10 p-8 text-center">
          <Zap size={48} className="mx-auto mb-4 text-amber-300" />
          <h2 className="text-xl font-semibold text-white mb-2">Storage Disabled</h2>
          <p className="text-slate-400 max-w-md mx-auto">
            Conversation storage is disabled. Enable storage in your gateway configuration to view conversation threads.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Conversations"
        subtitle={`${threads.length} conversation thread${threads.length !== 1 ? 's' : ''} recorded`}
        icon={MessageSquare}
        iconColor="text-sky-300"
        actions={
          <button
            type="button"
            onClick={handleRefresh}
            className="inline-flex items-center gap-2 rounded-xl border border-white/15 bg-slate-800/50 px-4 py-2 text-sm text-slate-200 transition-colors hover:border-white/30 hover:text-white"
          >
            <RefreshCcw size={16} />
            Refresh
          </button>
        }
      />

      {threadsError && (
        <div className="rounded-xl border border-amber-400/40 bg-amber-500/10 p-4 text-amber-100">
          {threadsError}
        </div>
      )}

      <div className="grid grid-cols-1 gap-5 lg:grid-cols-[400px_1fr]">
        {/* Thread List */}
        <div className="rounded-2xl border border-white/10 bg-slate-900/70">
          <div className="flex items-center justify-between border-b border-white/10 px-4 py-3">
            <div className="flex items-center gap-2 text-sm font-semibold text-white">
              <MessageSquare size={16} className="text-sky-300" />
              Threads
            </div>
            <Pill icon={MessageSquare} label={`${threads.length}`} />
          </div>

          <div className="max-h-[calc(100vh-380px)] min-h-[400px] space-y-2 overflow-y-auto p-3">
            {loadingThreads && <LoadingState message="Loading conversations..." />}

            {!loadingThreads && threads.length === 0 && (
              <EmptyState
                icon={MessageSquare}
                title="No conversations yet"
                description="Conversations will appear here as they are recorded"
              />
            )}

            {!loadingThreads &&
              threads.map((thread) => (
                <button
                  key={thread.id}
                  onClick={() => openThread(thread.id)}
                  className={`group flex w-full flex-col gap-2 rounded-xl border px-4 py-3 text-left transition ${
                    selectedThread?.id === thread.id
                      ? 'border-sky-400/50 bg-sky-500/10 shadow-[0_0_20px_rgba(14,165,233,0.1)]'
                      : 'border-white/10 bg-white/5 hover:border-white/20 hover:bg-white/10'
                  }`}
                >
                  <div className="flex items-center justify-between">
                    <div className="truncate text-sm font-semibold text-white">
                      {thread.metadata?.title || thread.id.slice(0, 16)}
                    </div>
                    <span className="flex-shrink-0 rounded-full bg-sky-500/20 px-2 py-0.5 text-xs font-medium text-sky-200">
                      {thread.message_count} msg
                    </span>
                  </div>

                  <div className="flex items-center gap-2 text-[11px] text-slate-400">
                    <Clock4 size={12} />
                    <span>{formatShortDate(thread.updated_at)}</span>
                  </div>

                  <div className="flex flex-wrap gap-1.5">
                    {thread.metadata?.frontdoor && (
                      <span className="rounded-md bg-slate-800/80 px-2 py-0.5 text-[10px] text-slate-300">
                        fd: {thread.metadata.frontdoor}
                      </span>
                    )}
                    {thread.metadata?.provider && (
                      <span className="rounded-md bg-slate-800/80 px-2 py-0.5 text-[10px] text-emerald-200">
                        prov: {thread.metadata.provider}
                      </span>
                    )}
                    {thread.metadata?.requested_model && (
                      <span className="rounded-md bg-slate-800/80 px-2 py-0.5 text-[10px] text-slate-300">
                        req: {thread.metadata.requested_model}
                      </span>
                    )}
                    {thread.metadata?.served_model && (
                      <span className="rounded-md bg-slate-800/80 px-2 py-0.5 text-[10px] text-emerald-200">
                        served: {thread.metadata.served_model}
                      </span>
                    )}
                    {thread.metadata?.app && (
                      <span className="rounded-md bg-slate-800/80 px-2 py-0.5 text-[10px] text-slate-200">
                        app: {thread.metadata.app}
                      </span>
                    )}
                    {thread.metadata?.stream === 'true' && (
                      <span className="rounded-md bg-amber-500/20 px-2 py-0.5 text-[10px] text-amber-200">
                        stream
                      </span>
                    )}
                    {thread.metadata?.topic && (
                      <span className="rounded-md bg-slate-800/80 px-2 py-0.5 text-[10px] text-slate-300">
                        {thread.metadata.topic}
                      </span>
                    )}
                  </div>
                </button>
              ))}
          </div>
        </div>

        {/* Thread Detail */}
        <div className="rounded-2xl border border-white/10 bg-slate-900/70">
          {!selectedThread && !loadingThreadDetail && (
            <div className="flex h-full min-h-[400px] flex-col items-center justify-center gap-4 text-slate-400">
              <MessageSquare size={48} className="text-slate-600" />
              <div className="text-center">
                <div className="text-sm font-medium">Select a thread</div>
                <div className="text-xs text-slate-500 mt-1">
                  Click on a conversation to view messages
                </div>
              </div>
            </div>
          )}

          {loadingThreadDetail && (
            <div className="flex h-full min-h-[400px] flex-col items-center justify-center gap-3 text-slate-300">
              <Loader2 className="h-8 w-8 animate-spin text-sky-400" />
              <div className="text-sm">Loading thread...</div>
            </div>
          )}

          {selectedThread && !loadingThreadDetail && (
            <div className="flex h-full flex-col">
              {/* Thread Header */}
              <div className="border-b border-white/10 px-5 py-4">
                <div className="flex flex-col gap-2">
                  <div className="text-lg font-semibold text-white">
                    {selectedThread.metadata?.title || selectedThread.id}
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <span className="inline-flex items-center gap-1.5 rounded-md bg-slate-800/80 px-2.5 py-1 text-xs text-slate-300">
                      <Clock4 size={12} />
                      {formatShortDate(selectedThread.created_at)}
                    </span>
                    <span className="inline-flex items-center gap-1.5 rounded-md bg-slate-800/80 px-2.5 py-1 text-xs text-slate-300">
                      <ServerCog size={12} />
                      {selectedThread.messages.length} messages
                    </span>
                    {selectedThread.metadata?.frontdoor && (
                      <span className="inline-flex items-center gap-1.5 rounded-md bg-slate-800/80 px-2.5 py-1 text-xs text-slate-300">
                        <Route size={12} />
                        fd: {selectedThread.metadata.frontdoor}
                      </span>
                    )}
                    {selectedThread.metadata?.provider && (
                      <span className="inline-flex items-center gap-1.5 rounded-md bg-slate-800/80 px-2.5 py-1 text-xs text-emerald-200">
                        <ServerCog size={12} />
                        provider: {selectedThread.metadata.provider}
                      </span>
                    )}
                    {selectedThread.metadata?.requested_model && (
                      <span className="inline-flex items-center gap-1.5 rounded-md bg-slate-800/80 px-2.5 py-1 text-xs text-slate-300">
                        <Signal size={12} />
                        req: {selectedThread.metadata.requested_model}
                      </span>
                    )}
                    {selectedThread.metadata?.served_model && (
                      <span className="inline-flex items-center gap-1.5 rounded-md bg-slate-800/80 px-2.5 py-1 text-xs text-emerald-200">
                        <Signal size={12} />
                        served: {selectedThread.metadata.served_model}
                      </span>
                    )}
                    {selectedThread.metadata?.app && (
                      <span className="inline-flex items-center gap-1.5 rounded-md bg-slate-800/80 px-2.5 py-1 text-xs text-slate-200">
                        <Compass size={12} />
                        app: {selectedThread.metadata.app}
                      </span>
                    )}
                    {selectedThread.metadata?.stream === 'true' && (
                      <span className="inline-flex items-center gap-1.5 rounded-md bg-amber-500/15 px-2.5 py-1 text-xs text-amber-200">
                        <Signal size={12} />
                        streamed
                      </span>
                    )}
                  </div>
                </div>
              </div>

              {/* Messages */}
              <div className="flex-1 space-y-4 overflow-y-auto px-5 py-4 max-h-[calc(100vh-480px)]">
                {selectedThread.messages.map((msg) => (
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
                {selectedThread.messages.length === 0 && (
                  <div className="text-sm text-slate-500 text-center py-8">
                    No messages in this thread.
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
