import { useNavigate, useLocation } from 'react-router-dom';
import { api } from '../api/client';
import { useAuthStore } from '../store/useAuthStore';
import { useToastStore } from '../store/useToastStore';
import { BrandHeader } from '../components/common/BrandHeader';
import { Button } from '../components/common/Button';
import { AppWrapper, FormCard, FormHeader } from '../components/layout/AppWrapper';
import { Styled } from './SetupRecoveryPage.Styled';

export function SetupRecoveryPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const words: string[] = (location.state as any)?.words || [];

  const fetchSession = useAuthStore((state) => state.fetchSession);
  const showToast = useToastStore((state) => state.showToast);

  const recoveryPhrase = words.join(' ');

  const handleCopy = () => {
    navigator.clipboard.writeText(recoveryPhrase).then(() => {
      showToast('Copied 12 recovery pebbles to clipboard!');
    });
  };

  const handleConfirm = async () => {
    try {
      await api.post('/api/setup/confirm', { acknowledged: true });
      await fetchSession();
      showToast('Vault active and secured!');
      navigate('/', { replace: true });
    } catch (_e) {
      await fetchSession();
      navigate('/', { replace: true });
    }
  };

  return (
    <AppWrapper size="default">
      <BrandHeader />

      <FormCard>
        <FormHeader>
          <h2>Your 12 Recovery Pebbles</h2>
          <p>Write these down in order and stash them somewhere secure.</p>
        </FormHeader>

        <Styled.WarningBox>
          <strong>⚠️ Critical Step:</strong> If you ever lose your master key, these 12 pebbles are the <em>only</em> way to reconstruct your zero-knowledge encryption key.
        </Styled.WarningBox>

        <Styled.WordsGrid>
          {words.map((word, index) => (
            <Styled.WordChip key={index}>
              <span className="num">{index + 1}.</span>
              <span className="word">{word}</span>
            </Styled.WordChip>
          ))}
        </Styled.WordsGrid>

        <Styled.ActionsRow>
          <Button variant="outline" onClick={handleCopy} style={{ flex: 1 }}>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
              <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
              <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
            </svg>
            Copy Phrase
          </Button>

          <Button variant="primary" onClick={handleConfirm} style={{ flex: 2 }}>
            I Stashed My Pebbles — Dive In
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" style={{ marginLeft: '4px' }}>
              <line x1="5" y1="12" x2="19" y2="12" />
              <polyline points="12 5 19 12 12 19" />
            </svg>
          </Button>
        </Styled.ActionsRow>
      </FormCard>
    </AppWrapper>
  );
}
