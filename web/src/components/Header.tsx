import { useAuth } from '../hooks/useAuth';

export default function Header() {
  const { user, logout } = useAuth();

  return (
    <header className="header">
      <span className="header-user">
        {user?.username} ({user?.role})
      </span>
      <button className="btn btn-secondary btn-sm" onClick={logout}>
        Logout
      </button>
    </header>
  );
}
