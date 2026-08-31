import { create } from 'zustand';

interface DeleteTokenModalState {
  isOpen: boolean;
  tokenId: string | null;
  issuer: string;
  accountName: string;
}

interface DeleteUserModalState {
  isOpen: boolean;
  userId: string | null;
  username: string;
}

interface ModalStore {
  deleteTokenModal: DeleteTokenModalState;
  deleteUserModal: DeleteUserModalState;
  openDeleteTokenModal: (tokenId: string, issuer: string, accountName: string) => void;
  closeDeleteTokenModal: () => void;
  openDeleteUserModal: (userId: string, username: string) => void;
  closeDeleteUserModal: () => void;
}

export const useModalStore = create<ModalStore>((set) => ({
  deleteTokenModal: {
    isOpen: false,
    tokenId: null,
    issuer: '',
    accountName: '',
  },
  deleteUserModal: {
    isOpen: false,
    userId: null,
    username: '',
  },
  openDeleteTokenModal: (tokenId, issuer, accountName) =>
    set({
      deleteTokenModal: { isOpen: true, tokenId, issuer, accountName },
    }),
  closeDeleteTokenModal: () =>
    set({
      deleteTokenModal: { isOpen: false, tokenId: null, issuer: '', accountName: '' },
    }),
  openDeleteUserModal: (userId, username) =>
    set({
      deleteUserModal: { isOpen: true, userId, username },
    }),
  closeDeleteUserModal: () =>
    set({
      deleteUserModal: { isOpen: false, userId: null, username: '' },
    }),
}));
