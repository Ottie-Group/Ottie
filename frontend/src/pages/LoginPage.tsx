import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '../store/useAuthStore';
import { useToastStore } from '../store/useToastStore';
import { BrandHeader } from '../components/common/BrandHeader';
import { Button } from '../components/common/Button';
import { FormGroup, Label, Input } from '../components/common/Input';
import { AppWrapper, FormCard, FormHeader } from '../components/layout/AppWrapper';
import { Styled } from './LoginPage.Styled';

export function LoginPage() {
  const navigate = useNavigate();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const login = useAuthStore((state) => state.login);
  const showToast = useToastStore((state) => state.showToast);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!username.trim() || !password.trim()) {
      showToast('Please enter your username and password.');
      return;
    }

    try {
      setIsLoading(true);
      const res = await login(username.trim(), password.trim());
      if (res.requires2FA) {
        navigate('/login/2fa', { state: { method: res.method } });
      } else if (res.requiresOTP) {
        navigate('/login/otp');
      } else {
        showToast('Welcome back to the den!');
        navigate('/', { replace: true });
      }
    } catch (err: any) {
      showToast(err.message || 'Login failed. Please check your credentials.');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <AppWrapper size="narrow">
      <BrandHeader />

      <FormCard>
        <FormHeader>
          <h2>Welcome Back to the Den</h2>
          <p>Enter your master key to open your fortified river dam.</p>
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
              autoFocus
            />
          </FormGroup>

          <FormGroup>
            <Label htmlFor="password">Master Key (Password)</Label>
            <Input
              id="password"
              type="password"
              placeholder="••••••••••••"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />
          </FormGroup>

          <Button type="submit" variant="primary" fullWidth disabled={isLoading}>
            {isLoading ? 'Diving In...' : 'Dive Into My Den'}
          </Button>

          <Styled.SubLink type="button" onClick={() => navigate('/recover')}>
            Lost your key? Rebuild your dam with 12 recovery pebbles
          </Styled.SubLink>
        </form>

        <Styled.InfoBox>
          <div>
            <h4>Fortified River Dam</h4>
            <p>Zero-knowledge encryption keeps your secrets completely safe & dry.</p>
          </div>
          <div>
            <svg width="36" height="36" viewBox="0 0 100 100" fill="none">
              <circle cx="50" cy="50" r="44" fill="#dcfce7" stroke="#16a34a" strokeWidth="3" />
              <path d="M32 44C34 40 40 40 42 44" stroke="#15803d" strokeWidth="4" strokeLinecap="round" />
              <path d="M58 44C60 40 66 40 68 44" stroke="#15803d" strokeWidth="4" strokeLinecap="round" />
              <path d="M44 58C47 62 53 62 56 58" stroke="#15803d" strokeWidth="3.5" strokeLinecap="round" />
            </svg>
          </div>
        </Styled.InfoBox>
      </FormCard>
    </AppWrapper>
  );
}
