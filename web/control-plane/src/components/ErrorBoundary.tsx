import { Component, type ReactNode, type ErrorInfo } from 'react';
import { AlertTriangle, RefreshCcw } from 'lucide-react';

interface Props {
    children: ReactNode;
    fallback?: ReactNode;
}

interface State {
    hasError: boolean;
    error: Error | null;
    errorInfo: ErrorInfo | null;
}

export class ErrorBoundary extends Component<Props, State> {
    constructor(props: Props) {
        super(props);
        this.state = { hasError: false, error: null, errorInfo: null };
    }

    static getDerivedStateFromError(error: Error): Partial<State> {
        return { hasError: true, error };
    }

    componentDidCatch(error: Error, errorInfo: ErrorInfo) {
        console.error('ErrorBoundary caught an error:', error, errorInfo);
        this.setState({ errorInfo });
    }

    handleReset = () => {
        this.setState({ hasError: false, error: null, errorInfo: null });
    };

    render() {
        if (this.state.hasError) {
            if (this.props.fallback) {
                return this.props.fallback;
            }

            return (
                <div className="flex min-h-[300px] flex-col items-center justify-center gap-4 p-8">
                    <div className="rounded-full bg-red-500/10 p-4">
                        <AlertTriangle size={32} className="text-red-400" />
                    </div>
                    <div className="text-center">
                        <h2 className="text-lg font-semibold text-white mb-2">Something went wrong</h2>
                        <p className="text-sm text-slate-400 max-w-md">
                            An unexpected error occurred. Please try refreshing or contact support if the problem persists.
                        </p>
                    </div>

                    {this.state.error && (
                        <div className="mt-4 max-w-lg w-full">
                            <details className="rounded-xl border border-red-500/20 bg-red-500/5 p-4">
                                <summary className="cursor-pointer text-sm font-medium text-red-300 hover:text-red-200">
                                    Error Details
                                </summary>
                                <div className="mt-3 space-y-2">
                                    <div className="text-xs text-red-200 font-mono break-all">
                                        {this.state.error.message}
                                    </div>
                                    {this.state.errorInfo?.componentStack && (
                                        <pre className="text-[10px] text-slate-400 font-mono overflow-auto max-h-40 p-2 bg-slate-950/50 rounded-sm">
                                            {this.state.errorInfo.componentStack}
                                        </pre>
                                    )}
                                </div>
                            </details>
                        </div>
                    )}

                    <button
                        onClick={this.handleReset}
                        className="mt-4 inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-violet-500/20 text-violet-200 hover:bg-violet-500/30 transition-colors"
                    >
                        <RefreshCcw size={14} />
                        Try Again
                    </button>
                </div>
            );
        }

        return this.props.children;
    }
}

// Higher-order component wrapper for functional components
export function withErrorBoundary<P extends object>(
    WrappedComponent: React.ComponentType<P>,
    fallback?: ReactNode
) {
    return function WithErrorBoundary(props: P) {
        return (
            <ErrorBoundary fallback={fallback}>
                <WrappedComponent {...props} />
            </ErrorBoundary>
        );
    };
}
