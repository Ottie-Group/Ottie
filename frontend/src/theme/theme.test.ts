import { describe, it, expect } from 'bun:test';
import { emeraldTheme, riverBlueTheme, themes } from './index';

describe('Emotion Theme Tokens', () => {
  it('should define emerald theme primary color palette', () => {
    expect(emeraldTheme.colors.primary).toBe('#059669');
    expect(emeraldTheme.colors.bgPage).toBe('#f6faf5');
    expect(emeraldTheme.colors.cardBg).toBe('#ffffff');
  });

  it('should define river blue theme primary color palette', () => {
    expect(riverBlueTheme.colors.primary).toBe('#0284c7');
    expect(riverBlueTheme.colors.bgPage).toBe('#f0f9ff');
    expect(riverBlueTheme.colors.primaryLight).toBe('#f0f9ff');
    expect(themes.river.name).toBe('river');
  });

  it('should define system font stacks and radii', () => {
    expect(emeraldTheme.fonts.body).toContain('BlinkMacSystemFont');
    expect(emeraldTheme.fonts.mono).toContain('ui-monospace');
    expect(emeraldTheme.radii.pill).toBe('9999px');
    expect(emeraldTheme.radii.lg).toBe('24px');
  });
});
