import styled from '@emotion/styled';

const GreetingBar = styled.div`
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: ${({ theme }) => theme.colors.cardBg};
  border: 1.5px solid ${({ theme }) => theme.colors.cardBorder};
  border-radius: ${({ theme }) => theme.radii.pill};
  padding: 10px 18px;
  box-shadow: ${({ theme }) => theme.shadows.sm};
  margin-bottom: 20px;
  flex-wrap: wrap;
  gap: 12px;

  @media (max-width: 540px) {
    flex-direction: column;
    align-items: stretch;
    border-radius: 20px;
    padding: 14px 16px;
    gap: 10px;
  }
`;

const UserInfo = styled.div`
  display: flex;
  align-items: center;
  gap: 10px;

  @media (max-width: 540px) {
    justify-content: space-between;
  }
`;

const AvatarBadge = styled.div`
  width: 38px;
  height: 38px;
  border-radius: 50%;
  background: ${({ theme }) => theme.colors.pastelGreen};
  border: 1.5px solid ${({ theme }) => theme.colors.pastelGreenBorder};
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;

  img {
    width: 24px;
    height: 24px;
  }
`;

const GreetingName = styled.span`
  font-size: 15px;
  font-weight: 700;
  color: ${({ theme }) => theme.colors.textDark};
`;

const RoleBadge = styled.span`
  background: #dcfce7;
  color: #15803d;
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.5px;
  padding: 2px 8px;
  border-radius: 6px;
  margin-left: 6px;
  text-transform: uppercase;
`;

const NavActions = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;

  @media (max-width: 540px) {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 6px;
    width: 100%;

    button {
      width: 100%;
      justify-content: center;
    }
  }
`;

const GoldBanner = styled.div`
  background: ${({ theme }) => theme.colors.pastelYellow};
  border: 1.5px solid ${({ theme }) => theme.colors.pastelYellowBorder};
  border-radius: ${({ theme }) => theme.radii.lg};
  padding: 16px 20px;
  margin-bottom: 20px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  box-shadow: ${({ theme }) => theme.shadows.sm};
  gap: 12px;

  @media (max-width: 480px) {
    border-radius: 18px;
    padding: 14px 16px;
  }
`;

const PurpleBanner = styled.div`
  background: ${({ theme }) => theme.colors.pastelPurple};
  border: 1.5px solid ${({ theme }) => theme.colors.pastelPurpleBorder};
  border-radius: ${({ theme }) => theme.radii.lg};
  padding: 16px 20px;
  margin-top: 24px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  box-shadow: ${({ theme }) => theme.shadows.sm};
  gap: 12px;

  @media (max-width: 480px) {
    border-radius: 18px;
    padding: 14px 16px;
  }
`;

const BannerContent = styled.div`
  h4 {
    font-size: 14px;
    font-weight: 800;
    color: #0f172a;
    margin-bottom: 4px;
  }
  p {
    font-size: 12px;
    color: #475569;
    margin-bottom: 8px;
  }
`;

const BannerLink = styled.button`
  background: none;
  border: none;
  color: #0f172a;
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0;
  text-decoration: none;

  &:hover {
    text-decoration: underline;
  }
`;

const SearchBox = styled.div`
  position: relative;
  margin-bottom: 20px;
`;

const SearchInput = styled.input`
  width: 100%;
  padding: 13px 20px 13px 44px;
  background: ${({ theme }) => theme.colors.cardBg};
  border: 1.5px solid ${({ theme }) => theme.colors.cardBorder};
  border-radius: ${({ theme }) => theme.radii.pill};
  font-size: 14px;
  outline: none;
  color: ${({ theme }) => theme.colors.textDark};
  box-shadow: ${({ theme }) => theme.shadows.sm};
  transition: all 0.15s ease;

  &:focus {
    border-color: ${({ theme }) => theme.colors.primary};
    box-shadow: 0 0 0 3px rgba(5, 150, 105, 0.15);
  }
`;

const SearchIcon = styled.svg`
  position: absolute;
  left: 16px;
  top: 50%;
  transform: translateY(-50%);
  color: ${({ theme }) => theme.colors.textDim};
  pointer-events: none;
`;

const CardsGrid = styled.div`
  display: flex;
  flex-direction: column;
  gap: 14px;
`;

const VaultCard = styled.div`
  background: ${({ theme }) => theme.colors.cardBg};
  border: 1.5px solid ${({ theme }) => theme.colors.cardBorder};
  border-radius: ${({ theme }) => theme.radii.lg};
  padding: 18px 20px;
  box-shadow: ${({ theme }) => theme.shadows.card};
  transition: transform 0.15s ease, box-shadow 0.15s ease;

  &:hover {
    box-shadow: ${({ theme }) => theme.shadows.hover};
  }
`;

const VaultCardHeader = styled.div`
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
`;

const IssuerBlock = styled.div`
  display: flex;
  align-items: center;
  gap: 12px;
`;

const IssuerDetails = styled.div`
  .issuer-title {
    font-size: 16px;
    font-weight: 800;
    color: ${({ theme }) => theme.colors.textDark};
    line-height: 1.2;
    display: flex;
    align-items: center;
  }
  .account-email {
    font-size: 12px;
    color: ${({ theme }) => theme.colors.textMuted};
    margin-top: 2px;
  }
`;

const CategoryBadge = styled.span`
  background: #f1f5f9;
  color: #475569;
  font-size: 10px;
  font-weight: 700;
  margin-left: 6px;
  padding: 2px 6px;
  border-radius: 4px;
`;

const DeleteButton = styled.button`
  background: none;
  border: none;
  color: ${({ theme }) => theme.colors.textDim};
  font-size: 16px;
  cursor: pointer;
  padding: 4px 6px;
  border-radius: 6px;
  transition: all 0.12s ease;

  &:hover {
    color: ${({ theme }) => theme.colors.danger};
    background: #fee2e2;
  }
`;

const CodeActionBox = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #f8faf7;
  border: 1.5px solid #eef2eb;
  border-radius: ${({ theme }) => theme.radii.md};
  padding: 10px 14px;
`;

const TotpDigits = styled.div`
  font-family: ${({ theme }) => theme.fonts.mono};
  font-size: 26px;
  font-weight: 700;
  letter-spacing: 3px;
  color: ${({ theme }) => theme.colors.primary};
  cursor: pointer;
  user-select: none;
  transition: transform 0.1s ease;

  &:hover {
    color: ${({ theme }) => theme.colors.primaryHover};
    transform: scale(1.02);
  }

  @media (max-width: 420px) {
    font-size: 20px;
    letter-spacing: 1.5px;
  }
`;

const CardActionsRight = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
`;

export const Styled = {
  GreetingBar,
  UserInfo,
  AvatarBadge,
  GreetingName,
  RoleBadge,
  NavActions,
  GoldBanner,
  PurpleBanner,
  BannerContent,
  BannerLink,
  SearchBox,
  SearchInput,
  SearchIcon,
  CardsGrid,
  VaultCard,
  VaultCardHeader,
  IssuerBlock,
  IssuerDetails,
  CategoryBadge,
  DeleteButton,
  CodeActionBox,
  TotpDigits,
  CardActionsRight,
};
