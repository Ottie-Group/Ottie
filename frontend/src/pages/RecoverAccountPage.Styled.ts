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

export const Styled = {
  BackNav,
  BackBtn,
};
