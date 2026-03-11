import './Dashboard.css';

interface Props {
  email: string | null;
  onLogout: () => void;
}

export function Dashboard({ email, onLogout }: Props) {
  return (
    <div className="dashboard">
      <div className="dashboard-card">
        <div className="dashboard-header">
          <h1>Welcome</h1>
          <button className="btn-logout" onClick={onLogout}>Sign Out</button>
        </div>
        <div className="dashboard-body">
          <div className="dashboard-avatar">
            {email?.[0]?.toUpperCase() || '?'}
          </div>
          <p className="dashboard-email">{email}</p>
          <p className="dashboard-note">You are authenticated successfully.</p>
        </div>
      </div>
    </div>
  );
}
