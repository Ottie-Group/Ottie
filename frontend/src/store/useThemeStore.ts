import { create } from 'zustand';
import { themes, ThemeName, OttieTheme } from '../theme';

interface ThemeState {
  themeName: ThemeName;
  currentTheme: OttieTheme;
  setTheme: (name: ThemeName) => void;
}

const STORAGE_KEY = 'ottie_user_theme';

export const useThemeStore = create<ThemeState>((set) => {
  let initialTheme: ThemeName = 'emerald';
  try {
    if (typeof localStorage !== 'undefined') {
      const saved = localStorage.getItem(STORAGE_KEY) as ThemeName | null;
      if (saved && themes[saved]) {
        initialTheme = saved;
      }
    }
  } catch (_e) {
    // Local storage might be restricted in some embedded environments
  }

  return {
    themeName: initialTheme,
    currentTheme: themes[initialTheme],
    setTheme: (name: ThemeName) => {
      if (themes[name]) {
        try {
          if (typeof localStorage !== 'undefined') {
            localStorage.setItem(STORAGE_KEY, name);
          }
        } catch (_e) {
          // Ignore
        }
        set({ themeName: name, currentTheme: themes[name] });
      }
    },
  };
});
