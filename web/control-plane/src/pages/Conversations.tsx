import React, { useEffect, useState } from 'react';
import { MessageSquare, User, Bot, Calendar } from 'lucide-react';

interface Thread {
    id: string;
    created_at: string;
    metadata?: {
        title?: string;
    };
}

interface Message {
    id: string;
    role: 'user' | 'assistant';
    content: string;
    created_at: string;
}

const Conversations = () => {
    const [threads, setThreads] = useState<Thread[]>([]);
    const [selectedThread, setSelectedThread] = useState<string | null>(null);
    const [messages, setMessages] = useState<Message[]>([]);
    const [loading, setLoading] = useState(true);

    // Fetch threads (mock for now as we don't have a list endpoint exposed easily without auth/tenant context in this MVP)
    // In a real implementation, we'd fetch from /responses/v1/threads if we had a list endpoint or similar.
    // For this MVP, we'll mock the list but try to fetch messages if a thread is selected.
    useEffect(() => {
        // Mock threads
        setThreads([
            { id: 'thread_1', created_at: new Date().toISOString(), metadata: { title: 'Project Planning' } },
            { id: 'thread_2', created_at: new Date(Date.now() - 86400000).toISOString(), metadata: { title: 'Code Review' } },
        ]);
        setLoading(false);
    }, []);

    const handleSelectThread = async (threadId: string) => {
        setSelectedThread(threadId);
        // In a real app, we would fetch messages here:
        // const res = await fetch(`/ responses / v1 / threads / ${ threadId }/messages`);
        // const data = await res.json();

        // Mock messages
        setMessages([
            { id: 'msg_1', role: 'user', content: 'How do I configure the gateway?', created_at: new Date().toISOString() },
            { id: 'msg_2', role: 'assistant', content: 'You can configure it using config.yaml...', created_at: new Date().toISOString() },
        ]);
    };

    return (
        <div className="flex h-[calc(100vh-8rem)] gap-6">
            {/* Thread List */}
            <div className="w-1/3 bg-gray-800 rounded-xl border border-gray-700 overflow-hidden flex flex-col">
                <div className="p-4 border-b border-gray-700 bg-gray-800">
                    <h2 className="font-semibold text-white">Conversations</h2>
                </div>
                <div className="flex-1 overflow-y-auto p-2 space-y-2">
                    {threads.map((thread) => (
                        <button
                            key={thread.id}
                            onClick={() => handleSelectThread(thread.id)}
                            className={`w-full text-left p-4 rounded-lg transition-colors ${selectedThread === thread.id
                                ? 'bg-blue-600 text-white'
                                : 'bg-gray-750 hover:bg-gray-700 text-gray-300'
                                }`}
                        >
                            <div className="font-medium truncate">{thread.metadata?.title || thread.id}</div>
                            <div className="text-xs opacity-70 mt-1 flex items-center gap-1">
                                <Calendar size={12} />
                                {new Date(thread.created_at).toLocaleDateString()}
                            </div>
                        </button>
                    ))}
                </div>
            </div>

            {/* Message View */}
            <div className="flex-1 bg-gray-800 rounded-xl border border-gray-700 overflow-hidden flex flex-col">
                {selectedThread ? (
                    <>
                        <div className="p-4 border-b border-gray-700 bg-gray-800 flex justify-between items-center">
                            <h2 className="font-semibold text-white">Thread: {selectedThread}</h2>
                            <span className="text-xs px-2 py-1 bg-green-900 text-green-300 rounded-full">Active</span>
                        </div>
                        <div className="flex-1 overflow-y-auto p-6 space-y-6">
                            {messages.map((msg) => (
                                <div key={msg.id} className={`flex gap-4 ${msg.role === 'user' ? 'justify-end' : ''}`}>
                                    <div className={`flex gap-3 max-w-[80%] ${msg.role === 'user' ? 'flex-row-reverse' : ''}`}>
                                        <div className={`w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0 ${msg.role === 'user' ? 'bg-blue-600' : 'bg-purple-600'
                                            }`}>
                                            {msg.role === 'user' ? <User size={16} /> : <Bot size={16} />}
                                        </div>
                                        <div className={`p-4 rounded-2xl ${msg.role === 'user'
                                            ? 'bg-blue-600 text-white rounded-tr-none'
                                            : 'bg-gray-700 text-gray-100 rounded-tl-none'
                                            }`}>
                                            <p className="text-sm leading-relaxed">{msg.content}</p>
                                            <div className="text-xs opacity-50 mt-2">
                                                {new Date(msg.created_at).toLocaleTimeString()}
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            ))}
                        </div>
                    </>
                ) : (
                    <div className="flex-1 flex items-center justify-center text-gray-500 flex-col gap-4">
                        <div className="p-4 bg-gray-700 rounded-full bg-opacity-50">
                            <MessageSquare size={48} />
                        </div>
                        <p>Select a conversation to view details</p>
                    </div>
                )}
            </div>
        </div>
    );
};

export default Conversations;
