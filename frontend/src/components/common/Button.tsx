import React from 'react';
import styled from '@emotion/styled';

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'danger' | 'outline' | 'pill';
  size?: 'sm' | 'md' | 'lg';
  fullWidth?: boolean;
}

export const Button = styled.button<ButtonProps>`
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: ${({ size }) => (size === 'sm' ? '6px 14px' : size === 'lg' ? '12px 24px' : '10px 18px')};
  border-radius: ${({ theme }) => theme.radii.pill};
  font-size: ${({ size }) => (size === 'sm' ? '12px' : size === 'lg' ? '14px' : '13px')};
  font-weight: 700;
  border: none;
  cursor: pointer;
  text-decoration: none;
  transition: all 0.15s ease;
  user-select: none;
  width: ${({ fullWidth }) => (fullWidth ? '100%' : 'auto')};

  &:disabled {
    opacity: 0.55;
    cursor: not-allowed;
    box-shadow: none !important;
  }

  ${({ variant, theme }) => {
    switch (variant) {
      case 'danger':
        return `
          background: ${theme.colors.danger};
          color: #ffffff;
          box-shadow: 0 4px 12px rgba(239, 68, 68, 0.25);
          &:hover:not(:disabled) {
            background: ${theme.colors.dangerHover};
            transform: translateY(-1px);
          }
        `;
      case 'outline':
        return `
          background: ${theme.colors.cardBg};
          border: 1.5px solid ${theme.colors.cardBorder};
          color: ${theme.colors.textDark};
          box-shadow: ${theme.shadows.sm};
          &:hover:not(:disabled) {
            background: #f8faf7;
            border-color: ${theme.colors.primaryBorder};
          }
        `;
      case 'primary':
      default:
        return `
          background: ${theme.colors.primary};
          color: #ffffff;
          box-shadow: 0 4px 12px rgba(5, 150, 105, 0.25);
          &:hover:not(:disabled) {
            background: ${theme.colors.primaryHover};
            transform: translateY(-1px);
          }
          &:active:not(:disabled) {
            transform: translateY(0);
          }
        `;
    }
  }}
`;
