import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '../store/useAuthStore';
import { useToastStore } from '../store/useToastStore';
import { BrandHeader } from '../components/common/BrandHeader';
import { Button } from '../components/common/Button';
import { FormGroup, Label, Input } from '../components/common/Input';
import { AppWrapper, FormCard, FormHeader } from '../components/layout/AppWrapper';
import { Styled } from './LoginOTPPage.Styled';

export function LoginOTPPage() {
  const navigate = useNavigate();
  const [code, setCode] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const verifyOTP = useAuthStore((state) => state.verifyOTP);
  const showToast = useToastStore((state) => state.showToast);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!code.trim()) {
      showToast('Please enter your emergency OTP code.');
      return;
    }

    try {
      setIsLoading(true);
      await verifyOTP(code.trim());
      showToast('Emergency OTP accepted!');
      navigate('/', { replace: true });
    } catch (err: any) {
      showToast(err.message || 'Invalid or used emergency OTP code.');
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
          <h2>Emergency OTP Access</h2>
          <p>Enter an emergency one-time passcode to bypass active 2FA checks.</p>
        </FormHeader>

        <form onSubmit={handleSubmit}>
          <FormGroup>
            <Label htmlFor="otp">Emergency One-Time Passcode</Label>
            <Input
              id="otp"
              type="text"
              isCode={true}
              placeholder="••••••••"
              value={code}
              onChange={(e) => setCode(e.target.value)}
              required
              autoFocus
            />
          </FormGroup>

          <Button type="submit" variant="primary" fullWidth disabled={isLoading}>
            {isLoading ? 'Authenticating...' : 'Authenticate with OTP'}
          </Button>
        </form>
      </FormCard>
    </AppWrapper>
  );
}
