import styled from '@emotion/styled';
import { useTheme } from '@emotion/react';

const HeaderContainer = styled.header`
  text-align: center;
  margin-bottom: 24px;
`;

const Title = styled.h1`
  font-size: 36px;
  font-weight: 800;
  color: ${({ theme }) => theme.colors.primary};
  display: inline-flex;
  align-items: center;
  gap: 8px;
  user-select: none;
`;

const MascotImg = styled.img`
  width: 38px;
  height: 38px;
  border-radius: 50%;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
`;

const RiverWaveAccent = styled.div`
  display: flex;
  justify-content: center;
  align-items: center;
  margin: 4px auto 0;
  height: 16px;
`;

export function BrandHeader() {
  const theme = useTheme();

  return (
    <HeaderContainer>
      <Title>
        Ottie <MascotImg src="/static/ottie.svg" alt="Ottie" />
      </Title>
      <RiverWaveAccent>
        <svg width="84" height="16" viewBox="0 0 84 16" fill="none" xmlns="http://www.w3.org/2000/svg">
          {/* Top River Wave */}
          <path
            d="M4 5C10 1 16 9 22 5C28 1 34 9 40 5C46 1 52 9 58 5C64 1 70 9 76 5C79 3 81 4 82 5"
            stroke={theme.colors.primary}
            strokeWidth="2.5"
            strokeLinecap="round"
          />
          {/* Bottom Ripple Wave */}
          <path
            d="M12 12C18 8 24 16 30 12C36 8 42 16 48 12C54 8 60 16 66 12C70 9 72 10.5 74 12"
            stroke={theme.colors.primaryBorder}
            strokeWidth="2.0"
            strokeLinecap="round"
          />
        </svg>
      </RiverWaveAccent>
    </HeaderContainer>
  );
}
