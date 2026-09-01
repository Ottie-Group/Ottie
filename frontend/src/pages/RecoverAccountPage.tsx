import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import { useToastStore } from '../store/useToastStore';
import { BrandHeader } from '../components/common/BrandHeader';
import { Button } from '../components/common/Button';
import { FormGroup, Label, Input, TextArea, HelpText } from '../components/common/Input';
import { AppWrapper, FormCard, FormHeader } from '../components/layout/AppWrapper';
import { Styled } from './RecoverAccountPage.Styled';

export function RecoverAccountPage() {
  const navigate = useNavigate();
  const [username, setUsername] = useState('');
  const [recoveryWords, setRecoveryWords] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const showToast = useToastStore((state) => state.showToast);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const trimmedUser = username.trim();
    const trimmedWords = recoveryWords.trim();
    if (!trimmedUser || !trimmedWords || !newPassword) {
      showToast('Please fill in all recovery fields.');
      return;
    }

    const wordsList = trimmedWords.split(/\s+/).filter(Boolean);
    if (wordsList.length !== 12) {
      showToast(`Expected 12 recovery words, but found ${wordsList.length}. Please check your phrase.`);
      return;
    }

    if (newPassword.length < 8) {
      showToast('New master key must be at least 8 characters long.');
      return;
    }

    if (newPassword !== confirmPassword) {
      showToast('New passwords do not match.');
      return;
    }

    try {
      setIsLoading(true);
      await api.post('/api/auth/recover', {
        username: username.trim(),
        recoveryKey: recoveryWords.trim(),
        newPassword,
      });
      showToast('Dam restored! Your master key has been reset.');
      navigate('/login', { replace: true });
    } catch (err: any) {
      showToast(err.message || 'Recovery failed. Invalid recovery phrase.');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <AppWrapper size="narrow">
      <BrandHeader />

      <Styled.BackNav>
        <Styled.BackBtn onClick={() => navigate('/login')}>
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
            <line x1="19" y1="12" x2="5" y2="12" />
            <polyline points="12 19 5 12 12 5" />
          </svg>
          Back to Login
        </Styled.BackBtn>
      </Styled.BackNav>

      <FormCard>
        <FormHeader>
          <h2>Recover Your River Dam</h2>
          <p>Provide your 12 recovery pebbles to reset your master key.</p>
        </FormHeader>

        <form onSubmit={handleSubmit}>
          <FormGroup>
            <Label htmlFor="username">Otter Name (Username)</Label>
            <Input
              id="username"
              type="text"
              placeholder="Your username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required
            />
          </FormGroup>

          <FormGroup>
            <Label htmlFor="recoveryWords">12 Recovery Words / Pebble Phrase</Label>
            <TextArea
              id="recoveryWords"
              placeholder="word1 word2 word3 word4 word5 word6 word7 word8 word9 word10 word11 word12"
              value={recoveryWords}
              onChange={(e) => setRecoveryWords(e.target.value)}
              required
            />
            <HelpText>Separate each word with a space.</HelpText>
          </FormGroup>

          <FormGroup>
            <Label htmlFor="newPassword">New Master Key (Password)</Label>
            <Input
              id="newPassword"
              type="password"
              placeholder="••••••••••••"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              required
            />
          </FormGroup>

          <FormGroup>
            <Label htmlFor="confirmPassword">Confirm New Master Key</Label>
            <Input
              id="confirmPassword"
              type="password"
              placeholder="••••••••••••"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              required
            />
          </FormGroup>

          <Button type="submit" variant="primary" fullWidth disabled={isLoading}>
            {isLoading ? 'Restoring Dam...' : 'Rebuild Dam & Reset Key'}
          </Button>
        </form>
      </FormCard>
    </AppWrapper>
  );
}
