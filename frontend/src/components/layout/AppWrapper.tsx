import styled from '@emotion/styled';

export const AppWrapper = styled.div<{ size?: 'narrow' | 'default' | 'wide' }>`
  width: 100%;
  margin: 0 auto;
  padding: 0 4px;
  max-width: ${({ size }) => {
    switch (size) {
      case 'narrow':
        return '440px';
      case 'wide':
        return '860px';
      case 'default':
      default:
        return '680px';
    }
  }};

  @media (max-width: 480px) {
    padding: 0;
  }
`;

export const FormCard = styled.div`
  background: ${({ theme }) => theme.colors.cardBg};
  border: 1.5px solid ${({ theme }) => theme.colors.cardBorder};
  border-radius: ${({ theme }) => theme.radii.lg};
  padding: 28px 24px;
  box-shadow: ${({ theme }) => theme.shadows.card};
  margin-bottom: 20px;

  @media (max-width: 480px) {
    padding: 20px 16px;
    border-radius: 18px;
  }
`;

export const FormHeader = styled.div`
  text-align: center;
  margin-bottom: 22px;

  h2 {
    font-size: 20px;
    font-weight: 800;
    color: ${({ theme }) => theme.colors.textDark};
    margin-bottom: 6px;
  }

  p {
    font-size: 13px;
    color: ${({ theme }) => theme.colors.textMuted};
  }
`;
