import styled from '@emotion/styled';

export const FormGroup = styled.div`
  margin-bottom: 18px;
  text-align: left;
`;

export const Label = styled.label`
  display: block;
  font-size: 13px;
  font-weight: 700;
  color: ${({ theme }) => theme.colors.textDark};
  margin-bottom: 6px;
`;

export const Input = styled.input<{ isCode?: boolean }>`
  width: 100%;
  padding: ${({ isCode }) => (isCode ? '14px 16px' : '12px 16px')};
  border-radius: ${({ theme }) => theme.radii.sm};
  background: #f8faf7;
  border: 1.5px solid ${({ theme }) => theme.colors.cardBorder};
  color: ${({ theme }) => theme.colors.textDark};
  font-size: ${({ isCode }) => (isCode ? '20px' : '14px')};
  font-family: ${({ isCode, theme }) => (isCode ? theme.fonts.mono : 'inherit')};
  letter-spacing: ${({ isCode }) => (isCode ? '4px' : 'normal')};
  text-align: ${({ isCode }) => (isCode ? 'center' : 'left')};
  outline: none;
  transition: all 0.15s ease;

  &:focus {
    background: #ffffff;
    border-color: ${({ theme }) => theme.colors.primary};
    box-shadow: 0 0 0 3px rgba(5, 150, 105, 0.15);
  }

  &:disabled {
    background-color: #f1f5f9;
    color: #94a3b8;
    border-color: #e2e8f0;
    cursor: not-allowed;
    opacity: 0.8;
  }
`;

export const TextArea = styled.textarea`
  width: 100%;
  padding: 12px 16px;
  border-radius: ${({ theme }) => theme.radii.sm};
  background: #f8faf7;
  border: 1.5px solid ${({ theme }) => theme.colors.cardBorder};
  color: ${({ theme }) => theme.colors.textDark};
  font-size: 13px;
  font-family: ${({ theme }) => theme.fonts.mono};
  line-height: 1.5;
  resize: vertical;
  min-height: 80px;
  outline: none;
  transition: all 0.15s ease;

  &:focus {
    background: #ffffff;
    border-color: ${({ theme }) => theme.colors.primary};
    box-shadow: 0 0 0 3px rgba(5, 150, 105, 0.15);
  }
`;

export const HelpText = styled.p`
  font-size: 12px;
  color: ${({ theme }) => theme.colors.textMuted};
  margin-top: 5px;
`;
