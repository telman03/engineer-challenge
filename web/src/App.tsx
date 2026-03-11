import { useAuth } from './hooks/useAuth';
import { LoginForm } from './components/LoginForm';
import { RegisterForm } from './components/RegisterForm';
import { ForgotPasswordForm } from './components/ForgotPasswordForm';
import { ResetPasswordForm } from './components/ResetPasswordForm';
import { Dashboard } from './components/Dashboard';

export default function App() {
  const auth = useAuth();

  switch (auth.view) {
    case 'login':
      return (
        <LoginForm
          loading={auth.loading}
          error={auth.error}
          onSubmit={auth.handleLogin}
          onSwitchToRegister={() => { auth.clearMessages(); auth.setView('register'); }}
          onForgotPassword={() => { auth.clearMessages(); auth.setView('forgot-password'); }}
        />
      );
    case 'register':
      return (
        <RegisterForm
          loading={auth.loading}
          error={auth.error}
          onSubmit={auth.handleRegister}
          onSwitchToLogin={() => { auth.clearMessages(); auth.setView('login'); }}
        />
      );
    case 'forgot-password':
      return (
        <ForgotPasswordForm
          loading={auth.loading}
          error={auth.error}
          success={auth.success}
          onSubmit={auth.handleForgotPassword}
          onSwitchToLogin={() => { auth.clearMessages(); auth.setView('login'); }}
        />
      );
    case 'reset-password':
      return (
        <ResetPasswordForm
          loading={auth.loading}
          error={auth.error}
          success={auth.success}
          defaultEmail={auth.resetEmail}
          onSubmit={auth.handleResetPassword}
          onSwitchToLogin={() => { auth.clearMessages(); auth.setView('login'); }}
        />
      );
    case 'dashboard':
      return <Dashboard email={auth.email} onLogout={auth.handleLogout} />;
  }
}
