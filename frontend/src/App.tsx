import { useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ThemeProvider } from '@emotion/react';
import { GlobalStyles } from './theme/GlobalStyles';
import { useAuthStore } from './store/useAuthStore';
import { useThemeStore } from './store/useThemeStore';
import { Toast } from './components/common/Toast';
import { ProtectedRoute, PublicOnlyRoute, SetupRoute, AdminRoute } from './routes/guards';

import { DashboardPage } from './pages/DashboardPage';
import { AddTokenPage } from './pages/AddTokenPage';
import { LoginPage } from './pages/LoginPage';
import { Login2FAPage } from './pages/Login2FAPage';
import { LoginOTPPage } from './pages/LoginOTPPage';
import { RecoverAccountPage } from './pages/RecoverAccountPage';
import { SetupWizardPage } from './pages/SetupWizardPage';
import { SetupRecoveryPage } from './pages/SetupRecoveryPage';
import { AdminPage } from './pages/AdminPage';
import { SettingsPage } from './pages/SettingsPage';

export function App() {
  const fetchSession = useAuthStore((state) => state.fetchSession);
  const currentTheme = useThemeStore((state) => state.currentTheme);

  useEffect(() => {
    fetchSession();
  }, [fetchSession]);

  return (
    <ThemeProvider theme={currentTheme}>
      <GlobalStyles />
      <Toast />
      <BrowserRouter>
        <Routes>
          {/* Protected Vault Routes */}
          <Route element={<ProtectedRoute />}>
            <Route path="/" element={<DashboardPage />} />
            <Route path="/add" element={<AddTokenPage />} />
            <Route path="/settings" element={<SettingsPage />} />
          </Route>

          {/* Admin Vault Routes */}
          <Route element={<AdminRoute />}>
            <Route path="/admin" element={<AdminPage />} />
          </Route>

          {/* Public / Auth Flow Routes */}
          <Route element={<PublicOnlyRoute />}>
            <Route path="/login" element={<LoginPage />} />
            <Route path="/login/2fa" element={<Login2FAPage />} />
            <Route path="/login/otp" element={<LoginOTPPage />} />
            <Route path="/recover" element={<RecoverAccountPage />} />
          </Route>

          {/* First-Time Setup Routes */}
          <Route element={<SetupRoute />}>
            <Route path="/setup" element={<SetupWizardPage />} />
            <Route path="/setup/recovery" element={<SetupRecoveryPage />} />
          </Route>

          {/* Catch-all Wildcard Route */}
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </ThemeProvider>
  );
}
