import { Routes, Route, Navigate } from 'react-router-dom';
import { useAuth } from './hooks/useAuth';
import Layout from './components/Layout';
import LoginForm from './components/LoginForm';
import ProtectedRoute from './components/ProtectedRoute';
import Dashboard from './components/Dashboard';
import ListenerList from './components/ListenerList';
import AgentList from './components/AgentList';
import AgentDetail from './components/AgentDetail';
import BuildPanel from './components/BuildPanel';
import AuditLog from './components/AuditLog';

export default function App() {
  const { isLoading } = useAuth();

  if (isLoading) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100vh' }}>
        <div className="spinner" />
      </div>
    );
  }

  return (
    <Routes>
      <Route path="/login" element={<LoginForm />} />
      <Route element={<ProtectedRoute />}>
        <Route element={<Layout />}>
          <Route path="/" element={<Dashboard />} />
          <Route path="/listeners" element={<ListenerList />} />
          <Route path="/agents" element={<AgentList />} />
          <Route path="/agents/:id" element={<AgentDetail />} />
          <Route path="/build" element={<BuildPanel />} />
          <Route path="/audit" element={<AuditLog />} />
        </Route>
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
