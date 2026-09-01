import styled from '@emotion/styled';
import { useToastStore } from '../../store/useToastStore';

const ToastContainer = styled.div`
  position: fixed;
  bottom: 24px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  flex-direction: column;
  gap: 8px;
  z-index: 2000;
  pointer-events: none;
`;

const ToastItem = styled.div`
  background: #0f172a;
  color: #ffffff;
  padding: 10px 18px;
  border-radius: ${({ theme }) => theme.radii.pill};
  font-size: 13px;
  font-weight: 700;
  box-shadow: 0 10px 25px -3px rgba(15, 23, 42, 0.25);
  display: inline-flex;
  align-items: center;
  gap: 8px;
  pointer-events: auto;
  animation: slideUp 0.2s cubic-bezier(0.16, 1, 0.3, 1);

  img {
    width: 22px;
    height: 22px;
    border-radius: 50%;
  }

  @keyframes slideUp {
    from {
      transform: translateY(12px);
      opacity: 0;
    }
    to {
      transform: translateY(0);
      opacity: 1;
    }
  }
`;

export function Toast() {
  const toasts = useToastStore((state) => state.toasts);

  if (toasts.length === 0) return null;

  return (
    <ToastContainer>
      {toasts.map((t) => (
        <ToastItem key={t.id}>
          <img src="/static/ottie.svg" alt="Ottie" />
          <span>{t.message}</span>
        </ToastItem>
      ))}
    </ToastContainer>
  );
}
