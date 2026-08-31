import { Navigate, Outlet } from 'react-router-dom';
import { useAuthStore } from '../store/useAuthStore';

function LoadingSplash() {
  return (
    <div style={{ marginTop: '120px', textAlign: 'center', color: '#059669', fontWeight: 800 }}>
      <img src="/static/ottie.svg" width="48" height="48" alt="Ottie" style={{ animation: 'bounce 1s infinite alternate' }} />
      <p style={{ marginTop: '12px' }}>Opening Ottie's Fortified Den...</p>
    </div>
  );
}

export function ProtectedRoute() {
  const { isAuthenticated, isLoading, setupNeeded } = useAuthStore();

  if (isLoading) return <LoadingSplash />;
  if (setupNeeded) return <Navigate to="/setup" replace />;
  if (!isAuthenticated) return <Navigate to="/login" replace />;

  return <Outlet />;
}

export function AdminRoute() {
  const { user, isAuthenticated, isLoading, setupNeeded } = useAuthStore();

  if (isLoading) return <LoadingSplash />;
  if (setupNeeded) return <Navigate to="/setup" replace />;
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  if (user?.role !== 'admin') return <Navigate to="/" replace />;

  return <Outlet />;
}

export function PublicOnlyRoute() {
  const { isAuthenticated, isLoading, setupNeeded } = useAuthStore();

  if (isLoading) return <LoadingSplash />;
  if (setupNeeded) return <Navigate to="/setup" replace />;
  if (isAuthenticated) return <Navigate to="/" replace />;

  return <Outlet />;
}

export function SetupRoute() {
  const { isLoading, setupNeeded } = useAuthStore();

  if (isLoading) return <LoadingSplash />;
  if (!setupNeeded) return <Navigate to="/" replace />;

  return <Outlet />;
}
