import React, { useEffect, useState } from 'react';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, LineChart, Line } from 'recharts';
import { Activity, Server, Cpu, Clock } from 'lucide-react';

interface MemoryStats {
    alloc: number;
    total_alloc: number;
    sys: number;
    num_gc: number;
}

interface Stats {
    uptime: string;
    go_version: string;
    num_goroutine: number;
    memory: MemoryStats;
}

const StatCard = ({ title, value, icon: Icon, color }: { title: string; value: string | number; icon: React.ElementType; color: string }) => (
    <div className="bg-gray-800 rounded-xl p-6 border border-gray-700">
        <div className="flex items-center justify-between mb-4">
            <h3 className="text-gray-400 font-medium">{title}</h3>
            <div className={`p-2 rounded-lg ${color} bg-opacity-20`}>
                <Icon size={20} className={color.replace('bg-', 'text-')} />
            </div>
        </div>
        <div className="text-3xl font-bold text-white">{value}</div>
    </div>
);

const Dashboard = () => {
    const [stats, setStats] = useState<Stats | null>(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        const fetchStats = async () => {
            try {
                const res = await fetch('/admin/api/stats');
                if (res.ok) {
                    const data = await res.json();
                    setStats(data);
                }
            } catch (error) {
                console.error("Failed to fetch stats", error);
            } finally {
                setLoading(false);
            }
        };

        fetchStats();
        const interval = setInterval(fetchStats, 5000);
        return () => clearInterval(interval);
    }, []);

    if (loading) return <div className="text-gray-400">Loading stats...</div>;
    if (!stats) return <div className="text-red-400">Failed to load stats</div>;

    // Mock data for charts (since backend only provides basic stats for now)
    const requestData = [
        { time: '10:00', reqs: 40 },
        { time: '10:05', reqs: 30 },
        { time: '10:10', reqs: 20 },
        { time: '10:15', reqs: 27 },
        { time: '10:20', reqs: 18 },
        { time: '10:25', reqs: 23 },
        { time: '10:30', reqs: 34 },
    ];

    return (
        <div className="space-y-6">
            <h1 className="text-2xl font-bold text-white">System Overview</h1>

            {/* Key Metrics */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                <StatCard
                    title="Uptime"
                    value={stats.uptime}
                    icon={Clock}
                    color="bg-blue-500"
                />
                <StatCard
                    title="Goroutines"
                    value={stats.num_goroutine}
                    icon={Activity}
                    color="bg-green-500"
                />
                <StatCard
                    title="Memory (Alloc)"
                    value={`${(stats.memory.alloc / 1024 / 1024).toFixed(2)} MB`}
                    icon={Server}
                    color="bg-purple-500"
                />
                <StatCard
                    title="GC Cycles"
                    value={stats.memory.num_gc}
                    icon={Cpu}
                    color="bg-orange-500"
                />
            </div>

            {/* Charts */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <div className="bg-gray-800 rounded-xl p-6 border border-gray-700">
                    <h3 className="text-lg font-medium text-white mb-6">Request Volume (Mock)</h3>
                    <div className="h-64">
                        <ResponsiveContainer width="100%" height="100%">
                            <BarChart data={requestData}>
                                <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
                                <XAxis dataKey="time" stroke="#9CA3AF" />
                                <YAxis stroke="#9CA3AF" />
                                <Tooltip
                                    contentStyle={{ backgroundColor: '#1F2937', borderColor: '#374151', color: '#F3F4F6' }}
                                />
                                <Bar dataKey="reqs" fill="#3B82F6" radius={[4, 4, 0, 0]} />
                            </BarChart>
                        </ResponsiveContainer>
                    </div>
                </div>

                <div className="bg-gray-800 rounded-xl p-6 border border-gray-700">
                    <h3 className="text-lg font-medium text-white mb-6">Latency (Mock)</h3>
                    <div className="h-64">
                        <ResponsiveContainer width="100%" height="100%">
                            <LineChart data={requestData}>
                                <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
                                <XAxis dataKey="time" stroke="#9CA3AF" />
                                <YAxis stroke="#9CA3AF" />
                                <Tooltip
                                    contentStyle={{ backgroundColor: '#1F2937', borderColor: '#374151', color: '#F3F4F6' }}
                                />
                                <Line type="monotone" dataKey="reqs" stroke="#10B981" strokeWidth={2} dot={false} />
                            </LineChart>
                        </ResponsiveContainer>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default Dashboard;
