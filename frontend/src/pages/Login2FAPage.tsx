import React, { useState } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { useAuthStore } from '../store/useAuthStore';
import { useToastStore } from '../store/useToastStore';
import { BrandHeader } from '../components/common/BrandHeader';
import { Button } from '../components/common/Button';
import { FormGroup, Label, Input } from '../components/common/Input';
import { AppWrapper, FormCard, FormHeader } from '../components/layout/AppWrapper';
import { Styled } from './Login2FAPage.Styled';

export function Login2FAPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const method = (location.state as any)?.method;

  const [code, setCode] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const verify2FA = useAuthStore((state) => state.verify2FA);
  const showToast = useToastStore((state) => state.showToast);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!code.trim()) {
      showToast('Please enter your 6-digit 2FA code.');
      return;
    }

    try {
      setIsLoading(true);
      await verify2FA(code.trim());
      showToast('2FA verified successfully!');
      navigate('/', { replace: true });
    } catch (err: any) {
      showToast(err.message || 'Invalid or expired 2FA code.');
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
          <h2>Two-Factor Verification</h2>
          <p>
            {method === 'email' || method === 'EMAIL'
              ? 'Enter the 6-digit passcode sent to your registered email address.'
              : 'Enter the 6-digit verification code sent via SMS.'}
          </p>
        </FormHeader>

        <form onSubmit={handleSubmit}>
          <FormGroup>
            <Label htmlFor="code">6-Digit Passcode</Label>
            <Input
              id="code"
              type="text"
              isCode={true}
              placeholder="123456"
              maxLength={6}
              value={code}
              onChange={(e) => setCode(e.target.value)}
              required
              autoFocus
            />
          </FormGroup>

          <Button type="submit" variant="primary" fullWidth disabled={isLoading}>
            {isLoading ? 'Verifying...' : 'Verify & Enter Den'}
          </Button>

          <Styled.SubLink type="button" onClick={() => navigate('/login/otp')}>
            Use emergency recovery phrase instead
          </Styled.SubLink>
        </form>
      </FormCard>
    </AppWrapper>
  );
}
