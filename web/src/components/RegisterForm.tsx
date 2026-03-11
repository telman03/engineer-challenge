import { useState, type FormEvent, useMemo } from 'react';
import { AuthLayout } from './AuthLayout';

interface Props {
  loading: boolean;
  error: string | null;
  onSubmit: (email: string, password: string) => void;
  onSwitchToLogin: () => void;
}

function getPasswordStrength(pw: string): { score: number; label: string } {
  let score = 0;
  if (pw.length >= 8) score++;
  if (/[A-Z]/.test(pw)) score++;
  if (/[a-z]/.test(pw)) score++;
  if (/[0-9]/.test(pw)) score++;
  if (/[^A-Za-z0-9]/.test(pw)) score++;

  if (score <= 2) return { score, label: 'Weak' };
  if (score <= 3) return { score, label: 'Fair' };
  if (score <= 4) return { score, label: 'Good' };
  return { score, label: 'Strong' };
}

export function RegisterForm({ loading, error, onSubmit, onSwitchToLogin }: Props) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [localError, setLocalError] = useState('');

  const strength = useMemo(() => getPasswordStrength(password), [password]);

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
    if (!/[A-Z]/.test(password) || !/[a-z]/.test(password) || !/[0-9]/.test(password) || !/[^A-Za-z0-9]/.test(password)) {
      setLocalError('Password must contain uppercase, lowercase, digit, and special character');
      return;
    }

    onSubmit(email, password);
  };

  const displayError = localError || error;

  return (
    <AuthLayout>
      <h1 className="auth-title">Create Account</h1>
      <p className="auth-subtitle">Get started with your new account.</p>

      {displayError && <div className="alert alert-error">{displayError}</div>}

      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <label className="form-label" htmlFor="reg-email">Email</label>
          <input
            id="reg-email"
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
          <label className="form-label" htmlFor="reg-password">Password</label>
          <input
            id="reg-password"
            className="form-input"
            type="password"
            value={password}
            onChange={e => setPassword(e.target.value)}
            placeholder="Min. 8 characters"
            required
            autoComplete="new-password"
          />
          {password && (
            <>
              <div className="password-strength">
                {[1, 2, 3, 4, 5].map(i => (
                  <div
                    key={i}
                    className={`strength-bar ${
                      i <= strength.score
                        ? strength.score <= 2
                          ? 'active'
                          : strength.score <= 4
                          ? 'medium'
                          : 'strong'
                        : ''
                    }`}
                  />
                ))}
              </div>
              <div className="strength-text">{strength.label}</div>
            </>
          )}
        </div>

        <div className="form-group">
          <label className="form-label" htmlFor="reg-confirm">Confirm Password</label>
          <input
            id="reg-confirm"
            className={`form-input ${confirm && password !== confirm ? 'error' : ''}`}
            type="password"
            value={confirm}
            onChange={e => setConfirm(e.target.value)}
            placeholder="Re-enter password"
            required
            autoComplete="new-password"
          />
          {confirm && password !== confirm && (
            <div className="form-error">Passwords do not match</div>
          )}
        </div>

        <button type="submit" className="btn btn-primary" disabled={loading}>
          {loading ? 'Creating account...' : 'Create Account'}
        </button>
      </form>

      <div className="auth-footer">
        Already have an account?{' '}
        <button type="button" className="auth-link" onClick={onSwitchToLogin}>
          Sign in
        </button>
      </div>
    </AuthLayout>
  );
}
