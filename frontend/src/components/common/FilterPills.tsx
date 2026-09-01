import styled from '@emotion/styled';

interface FilterPillsProps {
  categories: string[];
  selectedCategory: string;
  onSelectCategory: (category: string) => void;
}

const FilterStrip = styled.div`
  display: flex;
  gap: 8px;
  overflow-x: auto;
  padding-bottom: 4px;
  margin-bottom: 20px;
  user-select: none;

  &::-webkit-scrollbar {
    height: 4px;
  }
  &::-webkit-scrollbar-thumb {
    background: #cbd5e1;
    border-radius: 4px;
  }
`;

const PillChip = styled.button<{ isActive: boolean }>`
  padding: 6px 14px;
  border-radius: ${({ theme }) => theme.radii.pill};
  font-size: 13px;
  font-weight: 700;
  border: 1.5px solid ${({ isActive, theme }) => (isActive ? theme.colors.primary : theme.colors.cardBorder)};
  background: ${({ isActive, theme }) => (isActive ? theme.colors.primary : theme.colors.cardBg)};
  color: ${({ isActive, theme }) => (isActive ? '#ffffff' : theme.colors.textDark)};
  cursor: pointer;
  white-space: nowrap;
  box-shadow: ${({ theme }) => theme.shadows.sm};
  transition: all 0.15s ease;

  &:hover {
    ${({ isActive, theme }) =>
      !isActive &&
      `
      background: #f8faf7;
      border-color: ${theme.colors.primaryBorder};
    `}
  }
`;

export function FilterPills({ categories, selectedCategory, onSelectCategory }: FilterPillsProps) {
  const allCategories = ['all', ...categories.filter((c) => c.toLowerCase() !== 'all')];
  const current = (selectedCategory || 'all').toLowerCase();

  return (
    <FilterStrip>
      {allCategories.map((cat) => {
        const isActive = current === cat.toLowerCase();
        const label = cat === 'all' ? 'All Pebbles' : cat;

        return (
          <PillChip key={cat} isActive={isActive} onClick={() => onSelectCategory(cat)}>
            {label}
          </PillChip>
        );
      })}
    </FilterStrip>
  );
}
