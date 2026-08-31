import { describe, it, expect } from 'bun:test';
import { useToastStore } from './useToastStore';

describe('useToastStore', () => {
  it('should add and remove toast alerts', () => {
    const { showToast, removeToast } = useToastStore.getState();

    showToast('Token copied to clipboard!');
    const toasts = useToastStore.getState().toasts;
    expect(toasts.length).toBeGreaterThan(0);
    expect(toasts[toasts.length - 1].message).toBe('Token copied to clipboard!');

    const toastId = toasts[toasts.length - 1].id;
    removeToast(toastId);
    expect(useToastStore.getState().toasts.some((t) => t.id === toastId)).toBe(false);
  });
});
