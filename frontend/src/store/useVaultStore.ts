import { create } from 'zustand';
import { api } from '../api/client';

export interface AccountItem {
  id: string;
  issuer: string;
  accountName: string;
  category: string;
  createdAt?: string;
}

export interface LiveCodeItem {
  id: string;
  issuer: string;
  account_name: string;
  code: string;
  seconds_remaining: number;
}

interface VaultState {
  accounts: AccountItem[];
  liveCodes: Record<string, LiveCodeItem>;
  selectedCategory: string;
  searchQuery: string;
  isLoading: boolean;
  tickerActive: boolean;

  fetchAccounts: () => Promise<void>;
  fetchCodes: () => Promise<void>;
  startTicker: () => void;
  stopTicker: () => void;
  setCategory: (category: string) => void;
  setSearchQuery: (query: string) => void;
  addToken: (data: { secret: string; issuer: string; accountName: string; category: string; code?: string }) => Promise<void>;
  deleteToken: (id: string) => Promise<void>;
}

let tickerTimer: ReturnType<typeof setInterval> | null = null;

export const useVaultStore = create<VaultState>((set, get) => ({
  accounts: [],
  liveCodes: {},
  selectedCategory: 'all',
  searchQuery: '',
  isLoading: false,
  tickerActive: false,

  fetchAccounts: async () => {
    try {
      set({ isLoading: true });
      const res = await api.get('/api/accounts');
      set({ accounts: res.accounts || [], isLoading: false });
    } catch (_err) {
      set({ isLoading: false });
    }
  },

  fetchCodes: async () => {
    try {
      const res = await api.get<LiveCodeItem[]>('/api/codes');
      if (Array.isArray(res)) {
        const codeMap: Record<string, LiveCodeItem> = {};
        for (const item of res) {
          codeMap[item.id] = item;
        }
        set({ liveCodes: codeMap });
      }
    } catch (_err) {
      // Network hiccup - ignore and retry next interval
    }
  },

  startTicker: () => {
    if (tickerTimer) clearInterval(tickerTimer);
    get().fetchCodes();
    tickerTimer = setInterval(() => {
      get().fetchCodes();
    }, 1000);
    set({ tickerActive: true });
  },

  stopTicker: () => {
    if (tickerTimer) {
      clearInterval(tickerTimer);
      tickerTimer = null;
    }
    set({ tickerActive: false });
  },

  setCategory: (selectedCategory: string) => set({ selectedCategory }),
  setSearchQuery: (searchQuery: string) => set({ searchQuery }),

  addToken: async (data) => {
    await api.post('/api/tokens', data);
    await get().fetchAccounts();
    await get().fetchCodes();
  },

  deleteToken: async (id: string) => {
    await api.post('/api/tokens/delete', { id });
    set((state) => ({
      accounts: state.accounts.filter((a) => a.id !== id),
    }));
  },
}));
