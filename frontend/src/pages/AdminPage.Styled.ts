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

const UsersTable = styled.div`
  background: ${({ theme }) => theme.colors.cardBg};
  border: 1.5px solid ${({ theme }) => theme.colors.cardBorder};
  border-radius: ${({ theme }) => theme.radii.lg};
  overflow: hidden;
  box-shadow: ${({ theme }) => theme.shadows.card};
  margin-top: 24px;
`;

const TableHeader = styled.div`
  padding: 16px 20px;
  background: #f8faf7;
  border-bottom: 1.5px solid ${({ theme }) => theme.colors.cardBorder};
  display: flex;
  justify-content: space-between;
  align-items: center;

  h3 {
    font-size: 16px;
    font-weight: 800;
    color: ${({ theme }) => theme.colors.textDark};
  }
`;

const UserRow = styled.div`
  padding: 16px 20px;
  border-bottom: 1px solid #f1f5f9;
  display: flex;
  justify-content: space-between;
  align-items: center;

  &:last-child {
    border-bottom: none;
  }

  .user-meta {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .name {
    font-weight: 700;
    font-size: 15px;
    color: ${({ theme }) => theme.colors.textDark};
  }

  .count-chip {
    font-size: 11px;
    font-weight: 600;
    color: ${({ theme }) => theme.colors.textMuted};
    background: #f1f5f9;
    padding: 2px 8px;
    border-radius: 6px;
  }
`;

const RoleBadge = styled.span<{ role: string }>`
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.5px;
  padding: 2px 8px;
  border-radius: 6px;
  text-transform: uppercase;
  background: ${({ role }) => (role === 'admin' ? '#dcfce7' : '#e0e7ff')};
  color: ${({ role }) => (role === 'admin' ? '#15803d' : '#4338ca')};
`;

export const Styled = {
  BackNav,
  BackBtn,
  UsersTable,
  TableHeader,
  UserRow,
  RoleBadge,
};
