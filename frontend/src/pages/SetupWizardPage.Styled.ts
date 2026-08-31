import styled from '@emotion/styled';

const WelcomeNote = styled.div`
  background: ${({ theme }) => theme.colors.pastelGreen};
  border: 1.5px solid ${({ theme }) => theme.colors.pastelGreenBorder};
  border-radius: ${({ theme }) => theme.radii.md};
  padding: 14px 18px;
  margin-bottom: 20px;
  font-size: 13px;
  color: ${({ theme }) => theme.colors.pastelGreenText};
  line-height: 1.5;

  strong {
    color: #14532d;
  }
`;

export const Styled = {
  WelcomeNote,
};
