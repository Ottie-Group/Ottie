import styled from '@emotion/styled';

const WordsGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
  margin: 20px 0;

  @media (max-width: 480px) {
    grid-template-columns: repeat(2, 1fr);
  }
`;

const WordChip = styled.div`
  background: #f8faf7;
  border: 1.5px solid ${({ theme }) => theme.colors.cardBorder};
  border-radius: ${({ theme }) => theme.radii.sm};
  padding: 8px 10px;
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-family: ${({ theme }) => theme.fonts.mono};
  font-weight: 700;
  color: ${({ theme }) => theme.colors.textDark};

  .num {
    color: ${({ theme }) => theme.colors.textDim};
    font-size: 11px;
    width: 16px;
    text-align: right;
  }
  .word {
    color: ${({ theme }) => theme.colors.primary};
  }
`;

const WarningBox = styled.div`
  background: ${({ theme }) => theme.colors.pastelYellow};
  border: 1.5px solid ${({ theme }) => theme.colors.pastelYellowBorder};
  border-radius: ${({ theme }) => theme.radii.md};
  padding: 14px 18px;
  margin-bottom: 20px;
  font-size: 13px;
  color: ${({ theme }) => theme.colors.pastelYellowText};
  line-height: 1.5;

  strong {
    color: #713f12;
  }
`;

const ActionsRow = styled.div`
  display: flex;
  gap: 10px;
  margin-top: 20px;
`;

export const Styled = {
  WordsGrid,
  WordChip,
  WarningBox,
  ActionsRow,
};
