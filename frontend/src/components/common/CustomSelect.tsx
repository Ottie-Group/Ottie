import { useState, useRef, useEffect } from 'react';
import styled from '@emotion/styled';

export interface SelectOption {
  value: string;
  label: string;
  disabled?: boolean;
  disabledReason?: string;
}

interface CustomSelectProps {
  options: SelectOption[];
  value: string;
  onChange: (value: string) => void;
  name?: string;
  id?: string;
  disabled?: boolean;
}

const SelectWrapper = styled.div`
  position: relative;
  width: 100%;
  user-select: none;
`;

const SelectTrigger = styled.div<{ isOpen: boolean; disabled?: boolean }>`
  width: 100%;
  padding: 12px 38px 12px 14px;
  border-radius: ${({ theme }) => theme.radii.sm};
  background-color: ${({ disabled }) => (disabled ? '#f1f5f9' : '#f8faf7')};
  border: 1.5px solid ${({ isOpen, theme }) => (isOpen ? theme.colors.primary : theme.colors.cardBorder)};
  color: ${({ disabled, theme }) => (disabled ? '#94a3b8' : theme.colors.textDark)};
  font-size: 14px;
  font-weight: 600;
  cursor: ${({ disabled }) => (disabled ? 'not-allowed' : 'pointer')};
  display: flex;
  align-items: center;
  justify-content: space-between;
  box-shadow: ${({ isOpen }) => (isOpen ? '0 0 0 3px rgba(5, 150, 105, 0.15)' : 'none')};
  transition: all 0.15s ease;

  &:hover {
    ${({ disabled, theme }) => !disabled && `border-color: ${theme.colors.primaryBorder}; background-color: #ffffff;`}
  }
`;

const ArrowIcon = styled.svg<{ isOpen: boolean }>`
  position: absolute;
  right: 14px;
  top: 50%;
  transform: translateY(-50%) ${({ isOpen }) => (isOpen ? 'rotate(180deg)' : 'rotate(0deg)')};
  transition: transform 0.2s ease;
  pointer-events: none;
  stroke: ${({ theme }) => theme.colors.primary};
`;

const DropdownMenu = styled.div`
  position: absolute;
  top: calc(100% + 6px);
  left: 0;
  right: 0;
  background: #ffffff;
  border: 1.5px solid ${({ theme }) => theme.colors.primaryBorder};
  border-radius: ${({ theme }) => theme.radii.sm};
  box-shadow: 0 10px 25px -4px rgba(15, 23, 42, 0.12), 0 4px 6px -2px rgba(15, 23, 42, 0.04);
  z-index: 100;
  max-height: 240px;
  overflow-y: auto;
  padding: 6px;
`;

const OptionItem = styled.div<{ isSelected: boolean; isDisabled?: boolean }>`
  padding: 10px 12px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: ${({ isSelected }) => (isSelected ? '700' : '600')};
  color: ${({ isSelected, isDisabled, theme }) =>
    isDisabled ? theme.colors.textDim : isSelected ? theme.colors.primary : theme.colors.textDark};
  background: ${({ isSelected, isDisabled, theme }) => (isDisabled ? 'transparent' : isSelected ? theme.colors.primaryLight : 'transparent')};
  cursor: ${({ isDisabled }) => (isDisabled ? 'not-allowed' : 'pointer')};
  opacity: ${({ isDisabled }) => (isDisabled ? 0.6 : 1)};
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  transition: background 0.12s ease;

  &:hover {
    ${({ isDisabled, isSelected, theme }) =>
      !isDisabled && `background: ${isSelected ? theme.colors.primaryBorder : theme.colors.primaryLight};`}
  }
`;

const DisabledBadge = styled.span`
  font-size: 11px;
  font-weight: 600;
  color: #94a3b8;
  background: #f1f5f9;
  padding: 2px 6px;
  border-radius: 4px;
  margin-left: auto;
`;

const Checkmark = styled.svg`
  width: 14px;
  height: 14px;
  color: ${({ theme }) => theme.colors.primary};
`;

export function CustomSelect({ options, value, onChange, disabled }: CustomSelectProps) {
  const [isOpen, setIsOpen] = useState(false);
  const wrapperRef = useRef<HTMLDivElement>(null);

  const selectedOption = options.find((o) => o.value === value) || options[0];

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (wrapperRef.current && !wrapperRef.current.contains(e.target as Node)) {
        setIsOpen(false);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  return (
    <SelectWrapper ref={wrapperRef}>
      <SelectTrigger
        isOpen={isOpen}
        disabled={disabled}
        onClick={() => {
          if (!disabled) setIsOpen(!isOpen);
        }}
      >
        <span>{selectedOption ? selectedOption.label : 'Select...'}</span>
        <ArrowIcon isOpen={isOpen} width="16" height="16" viewBox="0 0 20 20" fill="none" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
          <path d="M6 8l4 4 4-4" />
        </ArrowIcon>
      </SelectTrigger>

      {isOpen && (
        <DropdownMenu>
          {options.map((opt) => {
            const isSelected = opt.value === value;
            return (
              <OptionItem
                key={opt.value}
                isSelected={isSelected}
                isDisabled={opt.disabled}
                onClick={() => {
                  if (opt.disabled) return;
                  onChange(opt.value);
                  setIsOpen(false);
                }}
              >
                <span>{opt.label}</span>
                {opt.disabled && (
                  <DisabledBadge>{opt.disabledReason || 'Disabled'}</DisabledBadge>
                )}
                {isSelected && !opt.disabled && (
                  <Checkmark viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                    <polyline points="20 6 9 17 4 12" />
                  </Checkmark>
                )}
              </OptionItem>
            );
          })}
        </DropdownMenu>
      )}
    </SelectWrapper>
  );
}
