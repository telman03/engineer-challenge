import { useState, type FormEvent } from 'react';
import { AuthLayout } from './AuthLayout';

interface Props {
  loading: boolean;
  error: string | null;
  success: string | null;
  onSubmit: (email: string) => void;
  onSwitchToLogin: () => void;
}

export function ForgotPasswordForm({ loading, error, success, onSubmit, onSwitchToLogin }: Props) {
  const [email, setEmail] = useState('');

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    onSubmit(email);
  };

  return (
    <AuthLayout>
      <h1 className="auth-title">Reset Password</h1>
      <p className="auth-subtitle">Enter your email and we'll send you a reset link.</p>

      {error && <div className="alert alert-error">{error}</div>}
      {success && <div className="alert alert-success">{success}</div>}

      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <label className="form-label" htmlFor="forgot-email">Email</label>
          <input
            id="forgot-email"
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

        <button type="submit" className="btn btn-primary" disabled={loading}>
          {loading ? 'Sending...' : 'Send Reset Link'}
        </button>
      </form>

      <div className="auth-footer">
        <button type="button" className="auth-link" onClick={onSwitchToLogin}>
          Back to Sign In
        </button>
      </div>
    </AuthLayout>
  );
}
