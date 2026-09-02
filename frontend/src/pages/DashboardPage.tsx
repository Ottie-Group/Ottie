import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '../store/useAuthStore';
import { useVaultStore } from '../store/useVaultStore';
import { useToastStore } from '../store/useToastStore';
import { useModalStore } from '../store/useModalStore';
import { BrandHeader } from '../components/common/BrandHeader';
import { Button } from '../components/common/Button';
import { ServiceIcon } from '../components/common/ServiceIcon';
import { CircularTimer } from '../components/common/CircularTimer';
import { FilterPills } from '../components/common/FilterPills';
import { Modal } from '../components/common/Modal';
import { AppWrapper } from '../components/layout/AppWrapper';
import { Styled } from './DashboardPage.Styled';

export function DashboardPage() {
  const navigate = useNavigate();
  const user = useAuthStore((state) => state.user);
  const logout = useAuthStore((state) => state.logout);
  const showToast = useToastStore((state) => state.showToast);

  const {
    accounts,
    liveCodes,
    selectedCategory,
    searchQuery,
    fetchAccounts,
    startTicker,
    stopTicker,
    setCategory,
    setSearchQuery,
    deleteToken,
  } = useVaultStore();

  const { deleteTokenModal, openDeleteTokenModal, closeDeleteTokenModal } = useModalStore();

  useEffect(() => {
    fetchAccounts();
    startTicker();
    return () => stopTicker();
  }, [fetchAccounts, startTicker, stopTicker]);

  // Extract unique non-empty categories present across user accounts
  const categories = Array.from(
    new Set(accounts.map((a) => (a.category || '').trim()).filter((c) => Boolean(c) && c.toLowerCase() !== 'all'))
  );

  // Filter accounts by search query & category
  const filteredAccounts = accounts.filter((account) => {
    const query = (searchQuery || '').trim().toLowerCase();
    const matchesSearch =
      !query ||
      (account.issuer || '').toLowerCase().includes(query) ||
      (account.accountName || '').toLowerCase().includes(query) ||
      (account.category || '').toLowerCase().includes(query);

    const isAll = !selectedCategory || selectedCategory.toLowerCase() === 'all';
    const accCategory = (account.category || '').trim().toLowerCase();
    const matchesCategory = isAll || accCategory === selectedCategory.toLowerCase();

    return matchesSearch && matchesCategory;
  });

  const handleCopyCode = (code: string) => {
    if (!code) return;
    navigator.clipboard.writeText(code).then(() => {
      showToast(`Copied code ${code} to clipboard!`);
    });
  };

  const handleLogout = async () => {
    await logout();
    showToast('Logged out of your den.');
    navigate('/login', { replace: true });
  };

  const handleDeleteConfirm = async () => {
    if (!deleteTokenModal.tokenId) return;
    try {
      await deleteToken(deleteTokenModal.tokenId);
      showToast(`Deleted ${deleteTokenModal.issuer} pebble.`);
      closeDeleteTokenModal();
    } catch {
      showToast('Failed to delete pebble.');
    }
  };

  return (
    <AppWrapper size="wide">
      <BrandHeader />

      {/* Greeting & Quick Bar */}
      <Styled.GreetingBar>
        <Styled.UserInfo>
          <Styled.AvatarBadge>
            <img src="/static/ottie.svg" alt="Avatar" />
          </Styled.AvatarBadge>
          <div>
            <Styled.GreetingName>
              Welcome, {user?.username || 'Otter'}
              {user?.role === 'admin' && <Styled.RoleBadge>Admin</Styled.RoleBadge>}
            </Styled.GreetingName>
          </div>
        </Styled.UserInfo>

        <Styled.NavActions>
          <Button variant="primary" size="sm" onClick={() => navigate('/add')}>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
              <line x1="12" y1="5" x2="12" y2="19" />
              <line x1="5" y1="12" x2="19" y2="12" />
            </svg>
            Add Pebble
          </Button>

          {user?.role === 'admin' && (
            <Button variant="outline" size="sm" onClick={() => navigate('/admin')}>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
                <circle cx="9" cy="7" r="4" />
                <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
                <path d="M16 3.13a4 4 0 0 1 0 7.75" />
              </svg>
              Admin
            </Button>
          )}

          <Button variant="outline" size="sm" onClick={() => navigate('/settings')}>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
              <circle cx="12" cy="12" r="3" />
              <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" />
            </svg>
            Settings
          </Button>

          <Button variant="danger" size="sm" onClick={handleLogout}>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
              <polyline points="16 17 21 12 16 7" />
              <line x1="21" y1="12" x2="9" y2="12" />
            </svg>
            Leave Den
          </Button>
        </Styled.NavActions>
      </Styled.GreetingBar>

      {/* Backup Advice Banner */}
      <Styled.GoldBanner>
        <Styled.BannerContent>
          <h4>Protect Your Secret Stash</h4>
          <p>
            Zero-knowledge pebbles are only safe if your master key or 12 recovery pebbles are kept safe.
          </p>
          <Styled.BannerLink onClick={() => navigate('/settings')}>
            View 2FA & Recovery Pebble Settings
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
              <line x1="5" y1="12" x2="19" y2="12" />
              <polyline points="12 5 19 12 12 19" />
            </svg>
          </Styled.BannerLink>
        </Styled.BannerContent>
        <div>
          <svg width="40" height="40" viewBox="0 0 100 100" fill="none">
            <circle cx="50" cy="50" r="44" fill="#fef08a" stroke="#ca8a04" strokeWidth="3" />
            <path d="M50 28V52" stroke="#854d0e" strokeWidth="5" strokeLinecap="round" />
            <circle cx="50" cy="68" r="4" fill="#854d0e" />
          </svg>
        </div>
      </Styled.GoldBanner>

      {/* Search Input */}
      <Styled.SearchBox>
        <Styled.SearchIcon width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
          <circle cx="11" cy="11" r="8" />
          <line x1="21" y1="21" x2="16.65" y2="16.65" />
        </Styled.SearchIcon>
        <Styled.SearchInput
          type="text"
          placeholder="Search your secret pebbles by service, email, or category..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
        />
      </Styled.SearchBox>

      {/* Category Pills Filter */}
      {categories.length > 0 && (
        <FilterPills
          categories={categories}
          selectedCategory={selectedCategory}
          onSelectCategory={setCategory}
        />
      )}

      {/* Token Cards Grid */}
      <Styled.CardsGrid>
        {filteredAccounts.length === 0 ? (
          <div style={{ textAlign: 'center', padding: '48px 20px', background: '#ffffff', borderRadius: '16px', border: '1.5px dashed #cbd5e1' }}>
            <svg width="48" height="48" viewBox="0 0 100 100" fill="none" style={{ margin: '0 auto 12px' }}>
              <circle cx="50" cy="50" r="44" fill="#f1f5f9" stroke="#94a3b8" strokeWidth="3" />
              <path d="M35 50H65M50 35V65" stroke="#64748b" strokeWidth="4" strokeLinecap="round" />
            </svg>
            <h3 style={{ fontSize: '16px', fontWeight: 800, color: '#1e293b', marginBottom: '4px' }}>
              {searchQuery || (selectedCategory && selectedCategory.toLowerCase() !== 'all') ? 'No Matching Pebbles Found' : 'Your Den is Empty'}
            </h3>
            <p style={{ fontSize: '13px', color: '#64748b', marginBottom: '16px' }}>
              {searchQuery || (selectedCategory && selectedCategory.toLowerCase() !== 'all')
                ? 'Try tweaking your search term or category filter.'
                : 'Start storing your 2FA secret pebbles in encrypted isolation.'}
            </p>
            <Button variant="primary" onClick={() => navigate('/add')}>
              Add Your First Pebble
            </Button>
          </div>
        ) : (
          filteredAccounts.map((account) => {
            const liveCode = liveCodes[account.id];
            const currentCode = liveCode?.code || '••••••';
            const remaining = liveCode?.seconds_remaining ?? 30;

            return (
              <Styled.VaultCard key={account.id}>
                <Styled.VaultCardHeader>
                  <Styled.IssuerBlock>
                    <ServiceIcon issuer={account.issuer} />
                    <Styled.IssuerDetails>
                      <div className="issuer-title">
                        {account.issuer}
                        {account.category && (
                          <Styled.CategoryBadge>{account.category}</Styled.CategoryBadge>
                        )}
                      </div>
                      <div className="account-email">{account.accountName}</div>
                    </Styled.IssuerDetails>
                  </Styled.IssuerBlock>

                  <Styled.DeleteButton
                    title="Delete Pebble"
                    onClick={() => openDeleteTokenModal(account.id, account.issuer, account.accountName)}
                  >
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
                      <polyline points="3 6 5 6 21 6" />
                      <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                    </svg>
                  </Styled.DeleteButton>
                </Styled.VaultCardHeader>

                <Styled.CodeActionBox>
                  <Styled.TotpDigits
                    title="Click to copy passcode"
                    onClick={() => handleCopyCode(currentCode)}
                  >
                    {currentCode.length === 6
                      ? `${currentCode.slice(0, 3)} ${currentCode.slice(3)}`
                      : currentCode}
                  </Styled.TotpDigits>

                  <Styled.CardActionsRight>
                    <CircularTimer secondsRemaining={remaining} />

                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleCopyCode(currentCode)}
                      title="Copy to Clipboard"
                    >
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
                        <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
                        <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
                      </svg>
                      Copy
                    </Button>
                  </Styled.CardActionsRight>
                </Styled.CodeActionBox>
              </Styled.VaultCard>
            );
          })
        )}
      </Styled.CardsGrid>

      {/* Export / Backup Feature Note */}
      <Styled.PurpleBanner>
        <Styled.BannerContent>
          <h4>Export Stash Backup</h4>
          <p>Download a decrypted JSON backup of all your secret pebbles anytime.</p>
          <Styled.BannerLink onClick={() => navigate('/settings')}>
            Manage Vault Backups
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
              <line x1="5" y1="12" x2="19" y2="12" />
              <polyline points="12 5 19 12 12 19" />
            </svg>
          </Styled.BannerLink>
        </Styled.BannerContent>
        <div>
          <svg width="40" height="40" viewBox="0 0 100 100" fill="none">
            <circle cx="50" cy="50" r="44" fill="#ede9fe" stroke="#7c3aed" strokeWidth="3" />
            <path d="M50 30V60M38 48L50 60L62 48" stroke="#5b21b6" strokeWidth="4" strokeLinecap="round" strokeLinejoin="round" />
            <path d="M30 70H70" stroke="#5b21b6" strokeWidth="4" strokeLinecap="round" />
          </svg>
        </div>
      </Styled.PurpleBanner>

      {/* Delete Token Confirmation Modal */}
      <Modal
        isOpen={deleteTokenModal.isOpen}
        title={`Delete "${deleteTokenModal.issuer}" Pebble?`}
        onClose={closeDeleteTokenModal}
        onConfirm={handleDeleteConfirm}
        isDanger={true}
        confirmText="Yes, Delete Pebble"
      >
        <p style={{ fontSize: '13px', color: '#475569', lineHeight: 1.5 }}>
          Are you sure you want to delete the secret pebble for <strong>{deleteTokenModal.issuer}</strong> ({deleteTokenModal.accountName})? This action cannot be undone.
        </p>
      </Modal>
    </AppWrapper>
  );
}
