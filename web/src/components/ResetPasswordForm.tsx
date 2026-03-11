import { useState, type FormEvent } from 'react';
import { AuthLayout } from './AuthLayout';

interface Props {
  loading: boolean;
  error: string | null;
  success: string | null;
  defaultEmail?: string;
  onSubmit: (email: string, token: string, newPassword: string) => void;
  onSwitchToLogin: () => void;
}

export function ResetPasswordForm({ loading, error, success, defaultEmail, onSubmit, onSwitchToLogin }: Props) {
  const [email, setEmail] = useState(defaultEmail ?? '');
  const [token, setToken] = useState('');
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [localError, setLocalError] = useState('');

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    setLocalError('');
    if (password !== confirm) {
      setLocalError('Passwords do not match');
      return;
    }
    if (password.length < 8) {
      setLocalError('Password must be at least 8 characters');
      return;
    }
    onSubmit(email, token, password);
  };

  const displayError = localError || error;

  return (
    <AuthLayout>
      <h1 className="auth-title">Set New Password</h1>
      <p className="auth-subtitle">Enter the reset token from your email and choose a new password.</p>

      {displayError && <div className="alert alert-error">{displayError}</div>}
      {success && <div className="alert alert-success">{success}</div>}

      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <label className="form-label" htmlFor="reset-email">Email</label>
          <input
            id="reset-email"
            className="form-input"
            type="email"
            value={email}
            onChange={e => setEmail(e.target.value)}
            placeholder="you@example.com"
            required
          />
        </div>

        <div className="form-group">
          <label className="form-label" htmlFor="reset-token">Reset Token</label>
          <input
            id="reset-token"
            className="form-input"
            type="text"
            value={token}
            onChange={e => setToken(e.target.value)}
            placeholder="Paste your reset token"
            required
            autoFocus
          />
        </div>

        <div className="form-group">
          <label className="form-label" htmlFor="reset-password">New Password</label>
          <input
            id="reset-password"
            className="form-input"
            type="password"
            value={password}
            onChange={e => setPassword(e.target.value)}
            placeholder="Min. 8 characters"
            required
            autoComplete="new-password"
          />
        </div>

        <div className="form-group">
          <label className="form-label" htmlFor="reset-confirm">Confirm Password</label>
          <input
            id="reset-confirm"
            className={`form-input ${confirm && password !== confirm ? 'error' : ''}`}
            type="password"
            value={confirm}
            onChange={e => setConfirm(e.target.value)}
            placeholder="Re-enter password"
            required
            autoComplete="new-password"
          />
        </div>

        <button type="submit" className="btn btn-primary" disabled={loading}>
          {loading ? 'Resetting...' : 'Reset Password'}
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
