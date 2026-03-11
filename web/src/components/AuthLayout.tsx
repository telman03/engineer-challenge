import type { ReactNode } from 'react';
import './AuthLayout.css';

interface Props {
  children: ReactNode;
}

export function AuthLayout({ children }: Props) {
  return (
    <div className="auth-layout">
      <div className="auth-card">
        <div className="auth-logo">
          <svg width="40" height="40" viewBox="0 0 40 40" fill="none">
            <rect width="40" height="40" rx="12" fill="#1a1a2e" />
            <path d="M12 20L20 12L28 20L20 28Z" fill="white" />
          </svg>
          <span className="auth-brand">Auth Service</span>
        </div>
        {children}
      </div>
    </div>
  );
}
