import { OttieTheme } from './types';

export const theme: OttieTheme = {
  colors: {
    bgPage: '#f6faf5',
    gridLine: '#e5ede3',
    primary: '#059669',
    primaryHover: '#047857',
    primaryLight: '#ecfdf5',
    primaryBorder: '#a7f3d0',
    cardBg: '#ffffff',
    cardBorder: '#e2e8f0',
    textDark: '#0f172a',
    textMuted: '#475569',
    textDim: '#94a3b8',
    danger: '#ef4444',
    dangerHover: '#dc2626',
    // Pastel Accent Banners
    pastelYellow: '#fef9c3',
    pastelYellowBorder: '#fde047',
    pastelYellowText: '#713f12',
    pastelPurple: '#ede9fe',
    pastelPurpleBorder: '#ddd6fe',
    pastelPurpleText: '#581c87',
    pastelGreen: '#dcfce7',
    pastelGreenBorder: '#86efac',
    pastelGreenText: '#14532d',
  },
  radii: {
    sm: '12px',
    md: '18px',
    lg: '24px',
    pill: '9999px',
  },
  shadows: {
    sm: '0 2px 4px rgba(15, 23, 42, 0.04)',
    card: '0 4px 20px -2px rgba(15, 23, 42, 0.06), 0 2px 4px -1px rgba(15, 23, 42, 0.03)',
    hover: '0 10px 25px -3px rgba(15, 23, 42, 0.08), 0 4px 6px -2px rgba(15, 23, 42, 0.04)',
    glow: '0 4px 14px rgba(5, 150, 105, 0.25)',
  },
  fonts: {
    body: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif',
    mono: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
  },
};
