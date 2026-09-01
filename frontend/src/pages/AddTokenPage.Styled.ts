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

const TabsWrap = styled.div`
  display: flex;
  background: #f1f5f9;
  padding: 4px;
  border-radius: ${({ theme }) => theme.radii.pill};
  margin-bottom: 22px;
`;

const TabBtn = styled.button<{ isActive: boolean }>`
  flex: 1;
  padding: 8px 16px;
  border-radius: ${({ theme }) => theme.radii.pill};
  font-size: 13px;
  font-weight: 700;
  border: none;
  background: ${({ isActive }) => (isActive ? '#ffffff' : 'transparent')};
  color: ${({ isActive, theme }) => (isActive ? theme.colors.primary : theme.colors.textMuted)};
  box-shadow: ${({ isActive }) => (isActive ? '0 2px 6px rgba(15, 23, 42, 0.08)' : 'none')};
  cursor: pointer;
  transition: all 0.15s ease;
`;

const DropZone = styled.div<{ isDragging: boolean }>`
  border: 2px dashed ${({ isDragging, theme }) => (isDragging ? theme.colors.primary : theme.colors.primaryBorder)};
  background: ${({ isDragging }) => (isDragging ? '#ecfdf5' : '#f8faf7')};
  border-radius: ${({ theme }) => theme.radii.md};
  padding: 32px 20px;
  text-align: center;
  cursor: pointer;
  transition: all 0.15s ease;
  margin-bottom: 20px;

  &:hover {
    background: #f0fdf4;
    border-color: ${({ theme }) => theme.colors.primary};
  }
`;

const CameraContainer = styled.div`
  position: relative;
  width: 100%;
  max-width: 420px;
  margin: 0 auto 20px;
  border-radius: ${({ theme }) => theme.radii.lg};
  overflow: hidden;
  background: #000000;
  box-shadow: ${({ theme }) => theme.shadows.card};
  border: 2px solid ${({ theme }) => theme.colors.primary};
`;

const VideoStream = styled.video`
  width: 100%;
  height: 280px;
  object-fit: cover;
  display: block;
`;

const ScanOverlay = styled.div`
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  pointer-events: none;
`;

const ScanBox = styled.div`
  width: 200px;
  height: 200px;
  border: 2px solid #34d399;
  border-radius: 16px;
  box-shadow: 0 0 0 9999px rgba(0, 0, 0, 0.45);
  position: relative;

  &::after {
    content: '';
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    height: 2px;
    background: #10b981;
    box-shadow: 0 0 8px #10b981;
    animation: scanLine 2s linear infinite;
  }

  @keyframes scanLine {
    0% {
      top: 0;
    }
    50% {
      top: 100%;
    }
    100% {
      top: 0;
    }
  }
`;

const CameraControls = styled.div`
  display: flex;
  justify-content: center;
  gap: 10px;
  margin-top: 10px;
`;

const LivePreviewCard = styled.div`
  background: #f0fdf4;
  border: 1.5px solid #86efac;
  border-radius: ${({ theme }) => theme.radii.md};
  padding: 16px 18px;
  margin-top: 12px;
  margin-bottom: 20px;
`;

const LivePreviewHeader = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
`;

const LivePreviewDigits = styled.div`
  font-family: 'JetBrains Mono', monospace, ui-monospace;
  font-size: 26px;
  font-weight: 800;
  color: #15803d;
  letter-spacing: 3px;
  display: flex;
  align-items: center;
  gap: 12px;
`;

const LiveBadge = styled.span`
  background: #dcfce7;
  color: #166534;
  font-size: 11px;
  font-weight: 700;
  padding: 3px 8px;
  border-radius: ${({ theme }) => theme.radii.pill};
  display: inline-flex;
  align-items: center;
  gap: 4px;
`;

const MatchBadge = styled.div<{ isMatch: boolean }>`
  font-size: 12px;
  font-weight: 700;
  margin-top: 6px;
  color: ${({ isMatch }) => (isMatch ? '#166534' : '#b91c1c')};
  display: flex;
  align-items: center;
  gap: 5px;
`;

export const Styled = {
  BackNav,
  BackBtn,
  TabsWrap,
  TabBtn,
  DropZone,
  CameraContainer,
  VideoStream,
  ScanOverlay,
  ScanBox,
  CameraControls,
  LivePreviewCard,
  LivePreviewHeader,
  LivePreviewDigits,
  LiveBadge,
  MatchBadge,
};
