import styled from '@emotion/styled';

const SubLink = styled.button`
  background: none;
  border: none;
  color: ${({ theme }) => theme.colors.primary};
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
  display: block;
  margin: 16px auto 0;
  text-align: center;

  &:hover {
    text-decoration: underline;
  }
`;

const InfoBox = styled.div`
  background: ${({ theme }) => theme.colors.pastelGreen};
  border: 1.5px solid ${({ theme }) => theme.colors.pastelGreenBorder};
  border-radius: ${({ theme }) => theme.radii.md};
  padding: 14px 18px;
  margin-top: 20px;
  display: flex;
  justify-content: space-between;
  align-items: center;

  h4 {
    font-size: 13px;
    font-weight: 800;
    color: ${({ theme }) => theme.colors.pastelGreenText};
    margin-bottom: 2px;
  }

  p {
    font-size: 12px;
    color: ${({ theme }) => theme.colors.pastelGreenText};
    opacity: 0.9;
  }
`;

export const Styled = {
  SubLink,
  InfoBox,
};
