import React, { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import { useAuthStore } from '../store/useAuthStore';
import { useToastStore } from '../store/useToastStore';
import { useModalStore } from '../store/useModalStore';
import { BrandHeader } from '../components/common/BrandHeader';
import { Button } from '../components/common/Button';
import { FormGroup, Label, Input } from '../components/common/Input';
import { CustomSelect } from '../components/common/CustomSelect';
import { Modal } from '../components/common/Modal';
import { AppWrapper, FormCard, FormHeader } from '../components/layout/AppWrapper';
import { Styled } from './AdminPage.Styled';

interface AdminUserItem {
  id: string;
  username: string;
  role: 'admin' | 'user';
  tokenCount: number;
  createdAt: string;
}

const ROLE_OPTIONS = [
  { value: 'user', label: 'Regular Otter (User)' },
  { value: 'admin', label: 'River Boss (Admin)' },
];

export function AdminPage() {
  const navigate = useNavigate();
  const currentUser = useAuthStore((state) => state.user);
  const showToast = useToastStore((state) => state.showToast);

  const [users, setUsers] = useState<AdminUserItem[]>([]);
  const [newUsername, setNewUsername] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [newRole, setNewRole] = useState('user');
  const [isCreating, setIsCreating] = useState(false);

  const { deleteUserModal, openDeleteUserModal, closeDeleteUserModal } = useModalStore();

  const fetchUsers = useCallback(async () => {
    try {
      const res = await api.get<{ users: AdminUserItem[] }>('/api/admin/users');
      setUsers(res.users || []);
    } catch {
      // Ignore
    }
  }, []);

  useEffect(() => {
    let isMounted = true;
    api.get<{ users: AdminUserItem[] }>('/api/admin/users').then((res) => {
      if (isMounted) setUsers(res.users || []);
    }).catch(() => {});
    return () => {
      isMounted = false;
    };
  }, []);

  const handleCreateUser = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newUsername.trim() || !newPassword) {
      showToast('Please provide username and password.');
      return;
    }

    try {
      setIsCreating(true);
      await api.post('/api/admin/users/create', {
        username: newUsername.trim(),
        password: newPassword,
        role: newRole,
      });
      showToast(`User ${newUsername.trim()} created!`);
      setNewUsername('');
      setNewPassword('');
      setNewRole('user');
      await fetchUsers();
    } catch (err: any) {
      showToast(err.message || 'Failed to create user.');
    } finally {
      setIsCreating(false);
    }
  };

  const handleDeleteUserConfirm = async () => {
    if (!deleteUserModal.userId) return;
    try {
      await api.post('/api/admin/users/delete', { id: deleteUserModal.userId });
      showToast(`User ${deleteUserModal.username} deleted.`);
      closeDeleteUserModal();
      await fetchUsers();
    } catch (err: any) {
      showToast(err.message || 'Failed to delete user.');
    }
  };

  return (
    <AppWrapper size="wide">
      <BrandHeader />

      <Styled.BackNav>
        <Styled.BackBtn onClick={() => navigate('/')}>
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
            <line x1="19" y1="12" x2="5" y2="12" />
            <polyline points="12 19 5 12 12 5" />
          </svg>
          Back to Dashboard
        </Styled.BackBtn>
      </Styled.BackNav>

      {/* Create New User Form Card */}
      <FormCard>
        <FormHeader>
          <h2>River Boss Administration</h2>
          <p>Create and manage user vaults across your Ottie instance.</p>
        </FormHeader>

        <form onSubmit={handleCreateUser}>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '12px' }}>
            <FormGroup>
              <Label htmlFor="createUsername">Otter Name (Username)</Label>
              <Input
                id="createUsername"
                type="text"
                placeholder="e.g. charlie"
                value={newUsername}
                onChange={(e) => setNewUsername(e.target.value)}
                required
              />
            </FormGroup>

            <FormGroup>
              <Label htmlFor="createPassword">Initial Master Key (Password)</Label>
              <Input
                id="createPassword"
                type="password"
                placeholder="••••••••••••"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                required
              />
            </FormGroup>

            <FormGroup>
              <Label>Assigned Role</Label>
              <CustomSelect
                options={ROLE_OPTIONS}
                value={newRole}
                onChange={(val) => setNewRole(val)}
              />
            </FormGroup>
          </div>

          <Button type="submit" variant="primary" disabled={isCreating}>
            {isCreating ? 'Provisioning Vault...' : 'Provision New Vault'}
          </Button>
        </form>
      </FormCard>

      {/* Users Table */}
      <Styled.UsersTable>
        <Styled.TableHeader>
          <h3>Active Vaults ({users.length})</h3>
          <Button variant="outline" size="sm" onClick={fetchUsers}>
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
              <polyline points="23 4 23 10 17 10" />
              <polyline points="1 20 1 14 7 14" />
              <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15" />
            </svg>
            Refresh
          </Button>
        </Styled.TableHeader>

        {users.map((u) => (
          <Styled.UserRow key={u.id}>
            <div className="user-meta">
              <div
                style={{
                  width: '36px',
                  height: '36px',
                  borderRadius: '50%',
                  background: '#f0fdf4',
                  border: '1.5px solid #bbf7d0',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontWeight: 800,
                  fontSize: '14px',
                  color: '#15803d',
                }}
              >
                {u.username.slice(0, 1).toUpperCase()}
              </div>

              <div>
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <span className="name">{u.username}</span>
                  <Styled.RoleBadge role={u.role}>{u.role}</Styled.RoleBadge>
                  {u.id === currentUser?.id && (
                    <span style={{ fontSize: '10px', color: '#059669', fontWeight: 700 }}>
                      (You)
                    </span>
                  )}
                </div>
                <div style={{ fontSize: '12px', color: '#64748b', marginTop: '2px' }}>
                  Created {new Date(u.createdAt).toLocaleDateString()}
                </div>
              </div>
            </div>

            <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
              <span className="count-chip">{u.tokenCount} pebbles</span>

              {u.id !== currentUser?.id && (
                <Button
                  variant="danger"
                  size="sm"
                  onClick={() => openDeleteUserModal(u.id, u.username)}
                >
                  Delete
                </Button>
              )}
            </div>
          </Styled.UserRow>
        ))}
      </Styled.UsersTable>

      {/* Delete User Modal */}
      <Modal
        isOpen={deleteUserModal.isOpen}
        title={`Delete User "${deleteUserModal.username}"?`}
        onClose={closeDeleteUserModal}
        onConfirm={handleDeleteUserConfirm}
        isDanger={true}
        confirmText="Yes, Delete Account"
      >
        <p style={{ fontSize: '13px', color: '#475569', lineHeight: 1.5 }}>
          Are you sure you want to delete <strong>{deleteUserModal.username}</strong>'s vault? This action cannot be undone.
        </p>
      </Modal>
    </AppWrapper>
  );
}
