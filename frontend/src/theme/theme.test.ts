import { describe, it, expect } from 'bun:test';
import { theme } from './index';

describe('Emotion Theme Tokens', () => {
  it('should define primary color palette', () => {
    expect(theme.colors.primary).toBe('#059669');
    expect(theme.colors.bgPage).toBe('#f6faf5');
    expect(theme.colors.cardBg).toBe('#ffffff');
    expect(theme.colors.pastelYellow).toBe('#fef9c3');
  });

  it('should define system font stacks and radii', () => {
    expect(theme.fonts.body).toContain('BlinkMacSystemFont');
    expect(theme.fonts.mono).toContain('ui-monospace');
    expect(theme.radii.pill).toBe('9999px');
    expect(theme.radii.lg).toBe('24px');
  });
});
