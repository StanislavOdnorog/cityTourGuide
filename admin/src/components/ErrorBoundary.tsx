import { Component } from 'react';
import type { ErrorInfo, ReactNode } from 'react';
import { useAuthStore } from '../store/authStore';

interface Props {
  children: ReactNode;
}

interface State {
  hasError: boolean;
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(): State {
    return { hasError: true };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('[ErrorBoundary]', error, info.componentStack);
  }

  private handleRetry = () => {
    this.setState({ hasError: false });
  };

  private handleGoToLogin = () => {
    // Navigate to login — works outside React Router context
    window.location.href = '/login';
  };

  render() {
    if (!this.state.hasError) {
      return this.props.children;
    }

    const isAuthenticated = useAuthStore.getState().isAuthenticated;

    return (
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          minHeight: '100vh',
          gap: 16,
          fontFamily: 'sans-serif',
        }}
      >
        <h1 style={{ fontSize: 24, margin: 0 }}>Something went wrong</h1>
        <p style={{ color: '#666', margin: 0 }}>
          An unexpected error occurred. You can try again or return to the login screen.
        </p>
        <div style={{ display: 'flex', gap: 12 }}>
          <button
            onClick={this.handleRetry}
            style={{
              padding: '8px 20px',
              fontSize: 14,
              cursor: 'pointer',
              borderRadius: 6,
              border: '1px solid #1677ff',
              background: '#1677ff',
              color: '#fff',
            }}
          >
            Try Again
          </button>
          {!isAuthenticated && (
            <button
              onClick={this.handleGoToLogin}
              style={{
                padding: '8px 20px',
                fontSize: 14,
                cursor: 'pointer',
                borderRadius: 6,
                border: '1px solid #d9d9d9',
                background: '#fff',
                color: '#333',
              }}
            >
              Go to Login
            </button>
          )}
        </div>
      </div>
    );
  }
}
