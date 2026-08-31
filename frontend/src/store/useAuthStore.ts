import { create } from 'zustand';
import { api } from '../api/client';

export interface User {
  id: string;
  username: string;
  role: 'admin' | 'user';
  has2FA?: boolean;
  deliveryMethod?: string;
  deliveryDest?: string;
  recoveryKey?: string;
}

export interface TwoFactorConfig {
  smtpConfigured: boolean;
  smsConfigured: boolean;
}

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  setupNeeded: boolean;
  csrfToken: string;
  twoFactorConfig: TwoFactorConfig;
  fetchSession: () => Promise<void>;
  login: (username: string, password: string) => Promise<{ requires2FA?: boolean; requiresOTP?: boolean; method?: string }>;
  verify2FA: (code: string) => Promise<void>;
  verifyOTP: (code: string) => Promise<void>;
  logout: () => Promise<void>;
  setUser: (user: User | null) => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  isAuthenticated: false,
  isLoading: true,
  setupNeeded: false,
  csrfToken: '',
  twoFactorConfig: {
    smtpConfigured: false,
    smsConfigured: false,
  },

  fetchSession: async () => {
    try {
      set({ isLoading: true });
      const res = await api.get('/api/me');
      if (res.csrfToken) {
        api.setCsrfToken(res.csrfToken);
      }
      set({
        user: res.user || null,
        isAuthenticated: !!res.authenticated,
        setupNeeded: !!res.setupNeeded,
        csrfToken: res.csrfToken || '',
        twoFactorConfig: res.twoFactorConfig || {
          smtpConfigured: false,
          smsConfigured: false,
        },
        isLoading: false,
      });
    } catch (_err) {
      set({
        user: null,
        isAuthenticated: false,
        isLoading: false,
      });
    }
  },

  login: async (username, password) => {
    const res = await api.post('/api/auth/login', { username, password });
    if (res.csrfToken) api.setCsrfToken(res.csrfToken);

    if (res.requires2FA || res.requiresOTP) {
      return res;
    }

    if (res.user) {
      set({
        user: res.user,
        isAuthenticated: true,
        setupNeeded: false,
      });
    }
    return res;
  },

  verify2FA: async (code) => {
    const res = await api.post('/api/auth/verify-2fa', { code });
    if (res.csrfToken) api.setCsrfToken(res.csrfToken);
    if (res.user) {
      set({
        user: res.user,
        isAuthenticated: true,
      });
    }
  },

  verifyOTP: async (code) => {
    const res = await api.post('/api/auth/verify-otp', { code });
    if (res.csrfToken) api.setCsrfToken(res.csrfToken);
    if (res.user) {
      set({
        user: res.user,
        isAuthenticated: true,
      });
    }
  },

  logout: async () => {
    try {
      await api.post('/api/auth/logout');
    } catch (_e) {
      // Ignore
    }
    set({
      user: null,
      isAuthenticated: false,
    });
  },

  setUser: (user) => set({ user, isAuthenticated: !!user }),
}));
