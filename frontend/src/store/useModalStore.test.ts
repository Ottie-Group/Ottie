import { describe, it, expect } from 'bun:test';
import { useModalStore } from './useModalStore';

describe('useModalStore', () => {
  it('should manage delete token modal state', () => {
    const { openDeleteTokenModal, closeDeleteTokenModal } = useModalStore.getState();

    openDeleteTokenModal('tok_123', 'GitHub', 'dev@ottie.local');
    expect(useModalStore.getState().deleteTokenModal.isOpen).toBe(true);
    expect(useModalStore.getState().deleteTokenModal.tokenId).toBe('tok_123');
    expect(useModalStore.getState().deleteTokenModal.issuer).toBe('GitHub');
    expect(useModalStore.getState().deleteTokenModal.accountName).toBe('dev@ottie.local');

    closeDeleteTokenModal();
    expect(useModalStore.getState().deleteTokenModal.isOpen).toBe(false);
    expect(useModalStore.getState().deleteTokenModal.tokenId).toBeNull();
  });

  it('should manage delete user modal state', () => {
    const { openDeleteUserModal, closeDeleteUserModal } = useModalStore.getState();

    openDeleteUserModal('usr_456', 'alice');
    expect(useModalStore.getState().deleteUserModal.isOpen).toBe(true);
    expect(useModalStore.getState().deleteUserModal.userId).toBe('usr_456');
    expect(useModalStore.getState().deleteUserModal.username).toBe('alice');

    closeDeleteUserModal();
    expect(useModalStore.getState().deleteUserModal.isOpen).toBe(false);
    expect(useModalStore.getState().deleteUserModal.userId).toBeNull();
  });
});
