import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import { useToastStore } from '../store/useToastStore';
import { BrandHeader } from '../components/common/BrandHeader';
import { Button } from '../components/common/Button';
import { FormGroup, Label, Input, HelpText } from '../components/common/Input';
import { AppWrapper, FormCard, FormHeader } from '../components/layout/AppWrapper';
import { Styled } from './SetupWizardPage.Styled';

export function SetupWizardPage() {
  const navigate = useNavigate();
  const [username, setUsername] = useState('admin');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const showToast = useToastStore((state) => state.showToast);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const trimmedUser = username.trim();
    if (!trimmedUser || !password) {
      showToast('Please provide an admin username and password.');
      return;
    }

    if (trimmedUser.length < 3) {
      showToast('Admin username must be at least 3 characters long.');
      return;
    }

    if (password.length < 8) {
      showToast('Master key must be at least 8 characters long.');
      return;
    }

    if (password !== confirmPassword) {
      showToast('Passwords do not match.');
      return;
    }

    try {
      setIsLoading(true);
      const res = await api.post('/api/setup', {
        username: username.trim(),
        password,
      });

      showToast('Vault created successfully!');
      navigate('/setup/recovery', { state: { words: res.words || [], recoveryKey: res.recoveryKey } });
    } catch (err: any) {
      showToast(err.message || 'Setup failed.');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <AppWrapper size="narrow">
      <BrandHeader />

      <FormCard>
        <FormHeader>
          <h2>Welcome to Ottie! 🦦</h2>
          <p>Let's build your fortified zero-knowledge 2FA river dam.</p>
        </FormHeader>

        <Styled.WelcomeNote>
          <strong>First-Time Setup:</strong> Create your primary administrator account. Your password encrypts your master data key with Argon2id zero-knowledge isolation.
        </Styled.WelcomeNote>

        <form onSubmit={handleSubmit}>
          <FormGroup>
            <Label htmlFor="adminUsername">Root Otter (Admin Username)</Label>
            <Input
              id="adminUsername"
              type="text"
              placeholder="e.g. admin"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required
              autoFocus
            />
          </FormGroup>

          <FormGroup>
            <Label htmlFor="adminPassword">Master Key (Password)</Label>
            <Input
              id="adminPassword"
              type="password"
              placeholder="••••••••••••"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />
            <HelpText>Choose a strong password to safeguard your 2FA seeds.</HelpText>
          </FormGroup>

          <FormGroup>
            <Label htmlFor="confirmAdminPassword">Confirm Master Key</Label>
            <Input
              id="confirmAdminPassword"
              type="password"
              placeholder="••••••••••••"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              required
            />
          </FormGroup>

          <Button type="submit" variant="primary" fullWidth disabled={isLoading}>
            {isLoading ? 'Creating Vault...' : 'Create Vault & Generate Pebbles'}
          </Button>
        </form>
      </FormCard>
    </AppWrapper>
  );
}
