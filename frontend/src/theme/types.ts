export type ThemeName = 'emerald' | 'river';

export interface OttieTheme {
  name: ThemeName;
  label: string;
  colors: {
    bgPage: string;
    gridLine: string;
    primary: string;
    primaryHover: string;
    primaryLight: string;
    primaryBorder: string;
    cardBg: string;
    cardBorder: string;
    textDark: string;
    textMuted: string;
    textDim: string;
    danger: string;
    dangerHover: string;
    // Pastel Accent Banners
    pastelYellow: string;
    pastelYellowBorder: string;
    pastelYellowText: string;
    pastelPurple: string;
    pastelPurpleBorder: string;
    pastelPurpleText: string;
    pastelGreen: string;
    pastelGreenBorder: string;
    pastelGreenText: string;
  };
  radii: {
    sm: string;
    md: string;
    lg: string;
    pill: string;
  };
  shadows: {
    sm: string;
    card: string;
    hover: string;
    glow: string;
  };
  fonts: {
    body: string;
    mono: string;
  };
}
