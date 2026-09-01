import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import { useAuthStore } from '../store/useAuthStore';
import { useToastStore } from '../store/useToastStore';
import { useThemeStore } from '../store/useThemeStore';
import { BrandHeader } from '../components/common/BrandHeader';
import { Button } from '../components/common/Button';
import { FormGroup, Label, Input, HelpText } from '../components/common/Input';
import { CustomSelect, SelectOption } from '../components/common/CustomSelect';
import { AppWrapper, FormCard, FormHeader } from '../components/layout/AppWrapper';
import { Styled } from './SettingsPage.Styled';

export function SettingsPage() {
  const navigate = useNavigate();
  const user = useAuthStore((state) => state.user);
  const twoFactorConfig = useAuthStore((state) => state.twoFactorConfig);
  const fetchSession = useAuthStore((state) => state.fetchSession);
  const showToast = useToastStore((state) => state.showToast);

  const themeName = useThemeStore((state) => state.themeName);
  const setTheme = useThemeStore((state) => state.setTheme);

  // Password state
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [isChangingPass, setIsChangingPass] = useState(false);

  // 2FA state
  const any2FAConfigured = twoFactorConfig.smtpConfigured || twoFactorConfig.smsConfigured;
  const initialMethod = user?.deliveryMethod === 'sms' && twoFactorConfig.smsConfigured
    ? 'sms'
    : twoFactorConfig.smtpConfigured
      ? 'email'
      : 'email';

  const [has2FA, setHas2FA] = useState(user?.has2FA && any2FAConfigured || false);
  const [deliveryMethod, setDeliveryMethod] = useState(user?.deliveryMethod || initialMethod);
  const [deliveryDest, setDeliveryDest] = useState(user?.deliveryDest || '');
  const [isSaving2FA, setIsSaving2FA] = useState(false);

  // Recovery Key
  const [recoveryKey, setRecoveryKey] = useState(user?.recoveryKey || '');
  const [isRegenerating, setIsRegenerating] = useState(false);

  const METHOD_OPTIONS: SelectOption[] = [
    {
      value: 'email',
      label: 'Email (SMTP)',
      disabled: !twoFactorConfig.smtpConfigured,
      disabledReason: !twoFactorConfig.smtpConfigured ? 'Not configured in .env' : undefined,
    },
    {
      value: 'sms',
      label: 'Text / Phone (SMS)',
      disabled: !twoFactorConfig.smsConfigured,
      disabledReason: !twoFactorConfig.smsConfigured ? 'Not configured in .env' : undefined,
    },
  ];

  const handlePasswordChange = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!currentPassword || !newPassword) {
      showToast('Please fill in both current and new password.');
      return;
    }
    if (newPassword.length < 8) {
      showToast('New master key must be at least 8 characters long.');
      return;
    }
    if (newPassword === currentPassword) {
      showToast('New master key cannot be the same as your current master key.');
      return;
    }
    if (newPassword !== confirmPassword) {
      showToast('New passwords do not match.');
      return;
    }

    try {
      setIsChangingPass(true);
      await api.post('/api/settings/password', {
        currentPassword,
        newPassword,
      });
      showToast('Master key updated successfully!');
      setCurrentPassword('');
      setNewPassword('');
      setConfirmPassword('');
    } catch (err: any) {
      showToast(err.message || 'Failed to update password.');
    } finally {
      setIsChangingPass(false);
    }
  };

  const handleSave2FA = async (e: React.FormEvent) => {
    e.preventDefault();
    if (has2FA) {
      if (deliveryMethod === 'email' && !twoFactorConfig.smtpConfigured) {
        showToast('Email (SMTP) is not configured on the server.');
        return;
      }
      if (deliveryMethod === 'sms' && !twoFactorConfig.smsConfigured) {
        showToast('Text/Phone (SMS) is not configured on the server.');
        return;
      }
      const trimmedDest = deliveryDest.trim();
      if (!trimmedDest) {
        showToast(
          deliveryMethod === 'email'
            ? 'Please provide a destination email address.'
            : 'Please provide a destination mobile phone number.'
        );
        return;
      }

      if (deliveryMethod === 'email') {
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        if (!emailRegex.test(trimmedDest)) {
          showToast('Please provide a valid email address (e.g. user@example.com).');
          return;
        }
      } else if (deliveryMethod === 'sms') {
        const phoneDigits = trimmedDest.replace(/\D/g, '');
        if (phoneDigits.length < 7 || phoneDigits.length > 18) {
          showToast('Please provide a valid mobile phone number (with at least 7 digits).');
          return;
        }
      }
    }

    try {
      setIsSaving2FA(true);
      await api.post('/api/settings/otp', {
        enabled: has2FA,
        method: deliveryMethod,
        destination: deliveryDest.trim(),
      });
      showToast('2FA preferences saved!');
      await fetchSession();
    } catch (err: any) {
      showToast(err.message || 'Failed to save 2FA settings.');
    } finally {
      setIsSaving2FA(false);
    }
  };

  const handleRegenerateRecovery = async () => {
    if (!window.confirm('Regenerate 12 recovery pebbles? Your previous recovery phrase will become invalid.')) {
      return;
    }

    try {
      setIsRegenerating(true);
      const res = await api.post<{ recoveryKey: string }>('/api/settings/recovery/regenerate');
      setRecoveryKey(res.recoveryKey || '');
      showToast('New 12 recovery pebbles generated!');
      await fetchSession();
    } catch (err: any) {
      showToast(err.message || 'Failed to regenerate recovery phrase.');
    } finally {
      setIsRegenerating(false);
    }
  };

  const handleExportBackup = async () => {
    try {
      const res = await api.post('/api/settings/export');
      const blob = new Blob([JSON.stringify(res, null, 2)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `ottie-vault-backup-${new Date().toISOString().slice(0, 10)}.json`;
      a.click();
      URL.revokeObjectURL(url);
      showToast('Vault backup downloaded!');
    } catch (err: any) {
      showToast(err.message || 'Export failed.');
    }
  };

  return (
    <AppWrapper size="default">
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

      <FormCard>
        <FormHeader>
          <h2>Vault Security & Preferences</h2>
          <p>Manage your master password, 2FA security, recovery key, and backups.</p>
        </FormHeader>

        {/* Change Master Key */}
        <form onSubmit={handlePasswordChange}>
          <Styled.SectionTitle>Change Master Key</Styled.SectionTitle>
          <FormGroup>
            <Label htmlFor="currentPass">Current Master Key</Label>
            <Input
              id="currentPass"
              type="password"
              placeholder="••••••••••••"
              value={currentPassword}
              onChange={(e) => setCurrentPassword(e.target.value)}
              required
            />
          </FormGroup>

          <Styled.FormGrid>
            <FormGroup>
              <Label htmlFor="newPass">New Master Key</Label>
              <Input
                id="newPass"
                type="password"
                placeholder="••••••••••••"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                required
              />
            </FormGroup>

            <FormGroup>
              <Label htmlFor="confirmPass">Confirm New Key</Label>
              <Input
                id="confirmPass"
                type="password"
                placeholder="••••••••••••"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                required
              />
            </FormGroup>
          </Styled.FormGrid>

          <Button type="submit" variant="primary" disabled={isChangingPass}>
            {isChangingPass ? 'Updating Key...' : 'Update Master Key'}
          </Button>
        </form>

        <Styled.SectionDivider />

        {/* 2FA Delivery Preferences */}
        <form onSubmit={handleSave2FA}>
          <Styled.SectionTitle>Two-Factor Authentication</Styled.SectionTitle>

          {!any2FAConfigured && (
            <Styled.UnconfiguredNotice>
              <strong>2FA Providers Not Configured:</strong> Neither Email (SMTP) nor Text/Phone (SMS) is configured in your server environment variables (`.env`). Configure `SMTP_HOST` or SMS credentials (`SMSGATE_LOGIN` or `TWILIO_ACCOUNT_SID`) to enable 2FA options.
            </Styled.UnconfiguredNotice>
          )}

          <Styled.CheckboxRow>
            <Styled.CheckboxInput
              type="checkbox"
              id="has2fa"
              checked={has2FA}
              disabled={!any2FAConfigured}
              onChange={(e) => setHas2FA(e.target.checked)}
            />
            <Styled.CheckboxLabel
              htmlFor="has2fa"
              isDisabled={!any2FAConfigured}
            >
              Enable Two-Factor Authentication on Login
            </Styled.CheckboxLabel>
          </Styled.CheckboxRow>

          {has2FA && any2FAConfigured && (
            <>
              <FormGroup>
                <Label>Delivery Method</Label>
                <CustomSelect
                  options={METHOD_OPTIONS}
                  value={deliveryMethod}
                  onChange={(val) => setDeliveryMethod(val)}
                />
              </FormGroup>

              {deliveryMethod === 'email' && (
                <FormGroup>
                  <Label htmlFor="emailDest">Destination Email Address</Label>
                  <Input
                    id="emailDest"
                    type="email"
                    placeholder="e.g. user@example.com"
                    value={deliveryDest}
                    onChange={(e) => setDeliveryDest(e.target.value)}
                    required
                  />
                  <HelpText>Login verification passcodes will be sent via configured SMTP.</HelpText>
                </FormGroup>
              )}

              {deliveryMethod === 'sms' && (
                <FormGroup>
                  <Label htmlFor="phoneDest">Destination Mobile / Phone Number</Label>
                  <Input
                    id="phoneDest"
                    type="tel"
                    placeholder="e.g. +1 (555) 234-5678"
                    value={deliveryDest}
                    onChange={(e) => setDeliveryDest(e.target.value)}
                    required
                  />
                  <HelpText>Login verification codes will be dispatched via SMS gateway.</HelpText>
                </FormGroup>
              )}
            </>
          )}

          <Button type="submit" variant="primary" disabled={isSaving2FA || (!has2FA && !user?.has2FA && !any2FAConfigured)}>
            {isSaving2FA ? 'Saving...' : 'Save 2FA Settings'}
          </Button>
        </form>

        <Styled.SectionDivider />

        {/* 12 Recovery Words */}
        <div>
          <Styled.SectionTitle>12 Recovery Words</Styled.SectionTitle>
          <Styled.SectionDesc>
            Your zero-knowledge recovery words for emergency key reconstruction.
          </Styled.SectionDesc>

          {recoveryKey ? (
            <Styled.RecoveryBox onClick={() => navigator.clipboard.writeText(recoveryKey).then(() => showToast('Copied recovery pebbles!'))}>
              {recoveryKey}
            </Styled.RecoveryBox>
          ) : (
            <p style={{ fontSize: '13px', color: '#94a3b8', margin: '12px 0' }}>No phrase generated yet.</p>
          )}

          <Button variant="outline" onClick={handleRegenerateRecovery} disabled={isRegenerating}>
            {isRegenerating ? 'Regenerating...' : 'Regenerate 12 Pebbles'}
          </Button>
        </div>

        <Styled.SectionDivider />

        {/* Export Decrypted Vault Backup */}
        <div>
          <Styled.SectionTitle>Stash Backup Export</Styled.SectionTitle>
          <Styled.SectionDesc>
            Download a decrypted JSON file containing all your secret pebbles and `otpauth://` URIs for portability.
          </Styled.SectionDesc>
          <Button variant="outline" onClick={handleExportBackup}>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" style={{ marginRight: '6px' }}>
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
              <polyline points="7 10 12 15 17 10" />
              <line x1="12" y1="15" x2="12" y2="3" />
            </svg>
            Export Stash Backup (JSON)
          </Button>
        </div>

        <Styled.SectionDivider />

        {/* Den Theme Appearance */}
        <div>
          <Styled.SectionTitle>Den Appearance & Color Theme</Styled.SectionTitle>
          <Styled.SectionDesc>
            Personalize your Ottie experience by switching between classic riverbank moss and fresh flowing river water themes.
          </Styled.SectionDesc>

          <Styled.ThemeGrid>
            <Styled.ThemeOptionCard
              isActive={themeName === 'emerald'}
              onClick={() => {
                setTheme('emerald');
                showToast('Switched to Moss Green theme!');
              }}
            >
              <Styled.ThemePreviewCircle color="#059669" border="#a7f3d0" />
              <div>
                <Styled.ThemeOptionTitle>Moss Green</Styled.ThemeOptionTitle>
                <Styled.ThemeOptionSub>Emerald riverbank green accent</Styled.ThemeOptionSub>
              </div>
              {themeName === 'emerald' && (
                <Styled.ThemeActiveCircle aria-label="Active theme">
                  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
                    <polyline points="20 6 9 17 4 12" />
                  </svg>
                </Styled.ThemeActiveCircle>
              )}
            </Styled.ThemeOptionCard>

            <Styled.ThemeOptionCard
              isActive={themeName === 'river'}
              onClick={() => {
                setTheme('river');
                showToast('Switched to River Blue theme!');
              }}
            >
              <Styled.ThemePreviewCircle color="#0284c7" border="#7dd3fc" />
              <div>
                <Styled.ThemeOptionTitle>River Blue</Styled.ThemeOptionTitle>
                <Styled.ThemeOptionSub>Vibrant blue and fresh river water</Styled.ThemeOptionSub>
              </div>
              {themeName === 'river' && (
                <Styled.ThemeActiveCircle aria-label="Active theme">
                  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
                    <polyline points="20 6 9 17 4 12" />
                  </svg>
                </Styled.ThemeActiveCircle>
              )}
            </Styled.ThemeOptionCard>
          </Styled.ThemeGrid>
        </div>
      </FormCard>
    </AppWrapper>
  );
}
