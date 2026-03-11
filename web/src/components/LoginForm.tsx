import { useState, type FormEvent } from 'react';
import { AuthLayout } from './AuthLayout';

interface Props {
  loading: boolean;
  error: string | null;
  onSubmit: (email: string, password: string) => void;
  onSwitchToRegister: () => void;
  onForgotPassword: () => void;
}

export function LoginForm({ loading, error, onSubmit, onSwitchToRegister, onForgotPassword }: Props) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    onSubmit(email, password);
  };

  return (
    <AuthLayout>
      <h1 className="auth-title">Sign In</h1>
      <p className="auth-subtitle">Welcome back. Enter your credentials to continue.</p>

      {error && <div className="alert alert-error">{error}</div>}

      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <label className="form-label" htmlFor="login-email">Email</label>
          <input
            id="login-email"
            className="form-input"
            type="email"
            value={email}
            onChange={e => setEmail(e.target.value)}
            placeholder="you@example.com"
            required
            autoComplete="email"
            autoFocus
          />
        </div>

        <div className="form-group">
          <label className="form-label" htmlFor="login-password">
            Password
            <button type="button" className="auth-link" style={{ float: 'right' }} onClick={onForgotPassword}>
              Forgot password?
            </button>
          </label>
          <input
            id="login-password"
            className="form-input"
            type="password"
            value={password}
            onChange={e => setPassword(e.target.value)}
            placeholder="Enter your password"
            required
            autoComplete="current-password"
          />
        </div>

        <button type="submit" className="btn btn-primary" disabled={loading}>
          {loading ? 'Signing in...' : 'Sign In'}
        </button>
      </form>

      <div className="auth-footer">
        Don't have an account?{' '}
        <button type="button" className="auth-link" onClick={onSwitchToRegister}>
          Create account
        </button>
      </div>
    </AuthLayout>
  );
}
