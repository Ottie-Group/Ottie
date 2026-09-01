import styled from '@emotion/styled';

const SERVICE_ICONS: Record<string, string> = {
  google: '/static/icons/google.svg',
  gmail: '/static/icons/google.svg',
  reddit: '/static/icons/reddit.svg',
  github: '/static/icons/github.svg',
  discord: '/static/icons/discord.svg',
  aws: '/static/icons/aws.svg',
  amazon: '/static/icons/aws.svg',
  microsoft: '/static/icons/microsoft.svg',
  azure: '/static/icons/microsoft.svg',
  outlook: '/static/icons/microsoft.svg',
  office365: '/static/icons/microsoft.svg',
  apple: '/static/icons/apple.svg',
  icloud: '/static/icons/apple.svg',
  twitter: '/static/icons/x.svg',
  'x.com': '/static/icons/x.svg',
  x: '/static/icons/x.svg',
  gitlab: '/static/icons/gitlab.svg',
  bitbucket: '/static/icons/bitbucket.svg',
  binance: '/static/icons/binance.svg',
  coinbase: '/static/icons/coinbase.svg',
  paypal: '/static/icons/paypal.svg',
  proton: '/static/icons/proton.svg',
  protonmail: '/static/icons/proton.svg',
  slack: '/static/icons/slack.svg',
  dropbox: '/static/icons/dropbox.svg',
  spotify: '/static/icons/spotify.svg',
  steam: '/static/icons/steam.svg',
  epic: '/static/icons/epicgames.svg',
  epicgames: '/static/icons/epicgames.svg',
  twitch: '/static/icons/twitch.svg',
  cloudflare: '/static/icons/cloudflare.svg',
  digitalocean: '/static/icons/digitalocean.svg',
  stripe: '/static/icons/stripe.svg',
  openai: '/static/icons/openai.svg',
  chatgpt: '/static/icons/openai.svg',
  facebook: '/static/icons/facebook.svg',
  meta: '/static/icons/facebook.svg',
  instagram: '/static/icons/instagram.svg',
  linkedin: '/static/icons/linkedin.svg',
};

const PASTEL_PALETTE = [
  { bg: '#dcfce7', text: '#15803d', border: '#bbf7d0' }, // Emerald
  { bg: '#e0e7ff', text: '#4338ca', border: '#c7d2fe' }, // Indigo
  { bg: '#fef3c7', text: '#b45309', border: '#fde68a' }, // Amber
  { bg: '#f3e8ff', text: '#7e22ce', border: '#e9d5ff' }, // Purple
  { bg: '#e0f2fe', text: '#0369a1', border: '#bae6fd' }, // Sky
  { bg: '#ffe4e6', text: '#be123c', border: '#fecdd3' }, // Rose
  { bg: '#f1f5f9', text: '#334155', border: '#cbd5e1' }, // Slate
];

function getMonogramColor(str: string) {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    hash = str.charCodeAt(i) + ((hash << 5) - hash);
  }
  const index = Math.abs(hash) % PASTEL_PALETTE.length;
  return PASTEL_PALETTE[index];
}

const IconContainer = styled.div<{ bgColor?: string; textColor?: string; borderColor?: string }>`
  width: 40px;
  height: 40px;
  border-radius: ${({ theme }) => theme.radii.sm};
  background: ${({ bgColor }) => bgColor || '#ffffff'};
  color: ${({ textColor }) => textColor || 'inherit'};
  border: 1.5px solid ${({ borderColor, theme }) => borderColor || theme.colors.cardBorder};
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  overflow: hidden;
  box-shadow: ${({ theme }) => theme.shadows.sm};
  font-weight: 800;
  font-size: 13px;
  letter-spacing: 0.5px;
  user-select: none;

  img {
    width: 24px;
    height: 24px;
    object-fit: contain;
    display: block;
  }
`;

export function ServiceIcon({ issuer }: { issuer: string }) {
  const cleanIssuer = (issuer || '').trim();
  const lower = cleanIssuer.toLowerCase();

  let matchIcon: string | null = null;
  for (const [key, path] of Object.entries(SERVICE_ICONS)) {
    if (lower.includes(key)) {
      matchIcon = path;
      break;
    }
  }

  if (matchIcon) {
    return (
      <IconContainer>
        <img src={matchIcon} alt={cleanIssuer} />
      </IconContainer>
    );
  }

  const initials = (cleanIssuer.length >= 2 ? cleanIssuer.substring(0, 2) : cleanIssuer.substring(0, 1) || 'OT').toUpperCase();
  const palette = getMonogramColor(cleanIssuer || 'Ottie');

  return (
    <IconContainer bgColor={palette.bg} textColor={palette.text} borderColor={palette.border}>
      <span>{initials}</span>
    </IconContainer>
  );
}
