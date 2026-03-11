import { useState, useCallback } from 'react';
import * as api from '../api/graphql';

export type AuthView = 'login' | 'register' | 'forgot-password' | 'reset-password' | 'dashboard';

interface AuthState {
  isAuthenticated: boolean;
  email: string | null;
  loading: boolean;
  error: string | null;
  success: string | null;
}

export function useAuth() {
  const [state, setState] = useState<AuthState>({
    isAuthenticated: !!localStorage.getItem('access_token'),
    email: localStorage.getItem('user_email'),
    loading: false,
    error: null,
    success: null,
  });
  const [view, setView] = useState<AuthView>(
    localStorage.getItem('access_token') ? 'dashboard' : 'login'
  );
  const [resetEmail, setResetEmail] = useState('');

  const clearMessages = useCallback(() => {
    setState(s => ({ ...s, error: null, success: null }));
  }, []);

  const handleLogin = useCallback(async (email: string, password: string) => {
    setState(s => ({ ...s, loading: true, error: null }));
    try {
      const result = await api.login(email, password);
      localStorage.setItem('access_token', result.accessToken);
      localStorage.setItem('refresh_token', result.refreshToken);
      localStorage.setItem('user_email', email);
      setState({ isAuthenticated: true, email, loading: false, error: null, success: null });
      setView('dashboard');
    } catch (err) {
      setState(s => ({ ...s, loading: false, error: (err as Error).message }));
    }
  }, []);

  const handleRegister = useCallback(async (email: string, password: string) => {
    setState(s => ({ ...s, loading: true, error: null }));
    try {
      const result = await api.register(email, password);
      localStorage.setItem('access_token', result.accessToken);
      localStorage.setItem('refresh_token', result.refreshToken);
      localStorage.setItem('user_email', email);
      setState({ isAuthenticated: true, email, loading: false, error: null, success: null });
      setView('dashboard');
    } catch (err) {
      setState(s => ({ ...s, loading: false, error: (err as Error).message }));
    }
  }, []);

  const handleForgotPassword = useCallback(async (email: string) => {
    setState(s => ({ ...s, loading: true, error: null, success: null }));
    try {
      const result = await api.requestPasswordReset(email);
      setResetEmail(email);
      const msg = result.token
        ? `Reset token: ${result.token}`
        : 'If an account with that email exists, a reset link has been sent.';
      setState(s => ({
        ...s,
        loading: false,
        success: msg,
      }));
      setTimeout(() => setView('reset-password'), 3000);
    } catch (err) {
      setState(s => ({ ...s, loading: false, error: (err as Error).message }));
    }
  }, []);

  const handleResetPassword = useCallback(async (email: string, token: string, newPassword: string) => {
    setState(s => ({ ...s, loading: true, error: null, success: null }));
    try {
      await api.resetPassword(email, token, newPassword);
      setState(s => ({
        ...s,
        loading: false,
        success: 'Password reset successfully. You can now log in.',
      }));
      setTimeout(() => setView('login'), 2000);
    } catch (err) {
      setState(s => ({ ...s, loading: false, error: (err as Error).message }));
    }
  }, []);

  const handleLogout = useCallback(() => {
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    localStorage.removeItem('user_email');
    setState({ isAuthenticated: false, email: null, loading: false, error: null, success: null });
    setView('login');
  }, []);

  return {
    ...state,
    view,
    setView,
    clearMessages,
    resetEmail,
    handleLogin,
    handleRegister,
    handleForgotPassword,
    handleResetPassword,
    handleLogout,
  };
}
