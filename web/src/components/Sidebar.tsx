import { NavLink } from 'react-router-dom';

export default function Sidebar() {
  const links = [
    { to: '/', label: 'Dashboard', icon: '◉' },
    { to: '/listeners', label: 'Listeners', icon: '◈' },
    { to: '/agents', label: 'Agents', icon: '⬡' },
    { to: '/build', label: 'Build', icon: '⚒' },
    { to: '/audit', label: 'Audit Log', icon: '☰' },
  ];

  return (
    <nav className="sidebar">
      <div className="sidebar-logo">◈ C2 Platform</div>
      <div className="sidebar-nav">
        {links.map((link) => (
          <NavLink
            key={link.to}
            to={link.to}
            end={link.to === '/'}
            className={({ isActive }) =>
              `sidebar-link${isActive ? ' active' : ''}`
            }
          >
            <span>{link.icon}</span>
            <span>{link.label}</span>
          </NavLink>
        ))}
      </div>
    </nav>
  );
}
