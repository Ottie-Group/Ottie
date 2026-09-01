import styled from '@emotion/styled';
import { useTheme } from '@emotion/react';

const CIRCLE_RADIUS = 14;
const CIRCLE_CIRCUMFERENCE = 2 * Math.PI * CIRCLE_RADIUS; // ~87.96

interface CircularTimerProps {
  secondsRemaining: number;
}

const TimerWrap = styled.div<{ isWarning: boolean; isDanger: boolean }>`
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: ${({ theme }) => theme.colors.cardBg};
  padding: 4px 8px;
  border-radius: ${({ theme }) => theme.radii.pill};
  border: 1px solid ${({ theme }) => theme.colors.cardBorder};
  user-select: none;
`;

const TimerSvg = styled.svg`
  transform: rotate(-90deg);
  width: 20px;
  height: 20px;
`;

const TimerBg = styled.circle`
  fill: none;
  stroke: #e2e8f0;
  stroke-width: 3.5;
`;

const TimerFg = styled.circle<{ strokeColor: string; offset: number }>`
  fill: none;
  stroke: ${({ strokeColor }) => strokeColor};
  stroke-width: 3.5;
  stroke-linecap: round;
  stroke-dasharray: ${CIRCLE_CIRCUMFERENCE};
  stroke-dashoffset: ${({ offset }) => offset};
  transition: stroke-dashoffset 0.8s linear, stroke 0.3s ease;
`;

const TimerNum = styled.span<{ textColor: string }>`
  font-family: ${({ theme }) => theme.fonts.mono};
  font-size: 11px;
  font-weight: 700;
  color: ${({ textColor }) => textColor};
  min-width: 16px;
  text-align: center;
`;

export function CircularTimer({ secondsRemaining }: CircularTimerProps) {
  const theme = useTheme();
  const isDanger = secondsRemaining <= 5;
  const isWarning = secondsRemaining <= 10 && !isDanger;

  let strokeColor = theme.colors.primary;
  let textColor = theme.colors.primary;

  if (isDanger) {
    strokeColor = '#ef4444';
    textColor = '#b91c1c';
  } else if (isWarning) {
    strokeColor = '#eab308';
    textColor = '#854d0e';
  }

  // Fraction of 30 seconds remaining
  const fraction = Math.max(0, Math.min(30, secondsRemaining)) / 30;
  const strokeOffset = CIRCLE_CIRCUMFERENCE * (1 - fraction);

  return (
    <TimerWrap isWarning={isWarning} isDanger={isDanger}>
      <TimerSvg viewBox="0 0 36 36">
        <TimerBg cx="18" cy="18" r={CIRCLE_RADIUS} />
        <TimerFg
          cx="18"
          cy="18"
          r={CIRCLE_RADIUS}
          strokeColor={strokeColor}
          offset={strokeOffset}
        />
      </TimerSvg>
      <TimerNum textColor={textColor}>
        {secondsRemaining > 0 ? secondsRemaining : '--'}
      </TimerNum>
    </TimerWrap>
  );
}
