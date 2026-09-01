import styled from '@emotion/styled';

const BackNav = styled.div`
  margin-bottom: 16px;
`;

const BackBtn = styled.button`
  background: none;
  border: none;
  color: ${({ theme }) => theme.colors.primary};
  font-size: 14px;
  font-weight: 700;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0;

  &:hover {
    text-decoration: underline;
  }
`;

const SectionDivider = styled.hr`
  border: none;
  border-top: 1.5px solid #f1f5f9;
  margin: 28px 0;
`;

const RecoveryBox = styled.div`
  background: #f0fdf4;
  border: 2px dashed ${({ theme }) => theme.colors.primaryBorder};
  border-radius: ${({ theme }) => theme.radii.md};
  padding: 16px;
  font-family: ${({ theme }) => theme.fonts.mono};
  font-size: 14px;
  font-weight: 700;
  color: ${({ theme }) => theme.colors.primary};
  text-align: center;
  user-select: all;
  cursor: pointer;
  margin: 14px 0;
`;

const UnconfiguredNotice = styled.div`
  background: ${({ theme }) => theme.colors.pastelYellow};
  border: 1.5px solid ${({ theme }) => theme.colors.pastelYellowBorder};
  border-radius: ${({ theme }) => theme.radii.md};
  padding: 12px 16px;
  font-size: 12px;
  color: ${({ theme }) => theme.colors.pastelYellowText};
  line-height: 1.5;
  margin-bottom: 16px;

  strong {
    color: #713f12;
  }
`;

const SectionTitle = styled.h3`
  font-size: 16px;
  font-weight: 800;
  margin-bottom: 14px;
  color: ${({ theme }) => theme.colors.textDark};
`;

const SectionDesc = styled.p`
  font-size: 13px;
  color: #64748b;
  margin-bottom: 14px;
`;

const CheckboxRow = styled.div`
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
`;

const CheckboxInput = styled.input`
  width: 18px;
  height: 18px;
`;

const CheckboxLabel = styled.label<{ isDisabled?: boolean }>`
  font-size: 14px;
  font-weight: 700;
  cursor: ${({ isDisabled }) => (isDisabled ? 'not-allowed' : 'pointer')};
  color: ${({ isDisabled }) => (isDisabled ? '#94a3b8' : 'inherit')};
`;

const FormGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 12px;
`;

const ThemeGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 14px;
  margin-top: 10px;
`;

const ThemeOptionCard = styled.div<{ isActive: boolean }>`
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  background: ${({ theme, isActive }) => (isActive ? theme.colors.primaryLight : '#ffffff')};
  border: 2px solid ${({ theme, isActive }) => (isActive ? theme.colors.primary : '#e2e8f0')};
  border-radius: ${({ theme }) => theme.radii.md};
  cursor: pointer;
  transition: all 0.15s ease;
  position: relative;

  &:hover {
    border-color: ${({ theme }) => theme.colors.primary};
    transform: translateY(-1px);
    box-shadow: ${({ theme }) => theme.shadows.sm};
  }
`;

const ThemePreviewCircle = styled.div<{ color: string; border: string }>`
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: ${({ color }) => color};
  border: 3px solid ${({ border }) => border};
  flex-shrink: 0;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.1);
`;

const ThemeOptionTitle = styled.div`
  font-size: 14px;
  font-weight: 800;
  color: ${({ theme }) => theme.colors.textDark};
`;

const ThemeOptionSub = styled.div`
  font-size: 12px;
  color: #64748b;
  margin-top: 2px;
`;

const ThemeActiveCircle = styled.div`
  margin-left: auto;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: ${({ theme }) => theme.colors.primary};
  color: #ffffff;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  box-shadow: 0 2px 5px rgba(0, 0, 0, 0.12);
`;

export const Styled = {
  BackNav,
  BackBtn,
  SectionDivider,
  RecoveryBox,
  UnconfiguredNotice,
  SectionTitle,
  SectionDesc,
  CheckboxRow,
  CheckboxInput,
  CheckboxLabel,
  FormGrid,
  ThemeGrid,
  ThemeOptionCard,
  ThemePreviewCircle,
  ThemeOptionTitle,
  ThemeOptionSub,
  ThemeActiveCircle,
};
