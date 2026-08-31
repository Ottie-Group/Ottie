// Toast Notification Manager
function showToast(message, duration = 2200) {
  let container = document.getElementById('toast-container');
  if (!container) {
    container = document.createElement('div');
    container.id = 'toast-container';
    container.className = 'toast-container';
    document.body.appendChild(container);
  }

  const toast = document.createElement('div');
  toast.className = 'toast';
  toast.innerHTML = `<img src="/static/ottie.svg" width="24" height="24" style="border-radius:50%"> <span>${message}</span>`;
  container.appendChild(toast);

  setTimeout(() => {
    toast.style.opacity = '0';
    toast.style.transform = 'translateY(-10px)';
    toast.style.transition = 'all 0.2s ease';
    setTimeout(() => toast.remove(), 200);
  }, duration);
}

// Copy Code to Clipboard
function copyToClipboard(rawCode, issuer) {
  const cleanCode = (rawCode || '').replace(/\s+/g, '');
  if (!cleanCode || cleanCode === '------') return;

  navigator.clipboard.writeText(cleanCode).then(() => {
    showToast(`Copied ${cleanCode} for ${issuer || 'token'}!`);
  }).catch(() => {
    const tempInput = document.createElement('input');
    tempInput.value = cleanCode;
    document.body.appendChild(tempInput);
    tempInput.select();
    document.execCommand('copy');
    document.body.removeChild(tempInput);
    showToast(`Copied ${cleanCode}!`);
  });
}

// Real-time Live TOTP Codes & Circular Countdown Ticker
const CIRCLE_RADIUS = 14;
const CIRCLE_CIRCUMFERENCE = 2 * Math.PI * CIRCLE_RADIUS; // ~87.96

async function refreshCodes() {
  const accountsContainer = document.getElementById('accounts-list');
  if (!accountsContainer) return;

  try {
    const res = await fetch('/api/codes', { credentials: 'same-origin' });
    if (!res.ok) return;
    const codes = await res.json();

    for (const c of codes) {
      const codeEl = document.getElementById('code-' + c.id);
      const timerContainer = document.getElementById('timer-box-' + c.id);
      const timerText = document.getElementById('timer-text-' + c.id);
      const timerProgress = document.getElementById('timer-progress-' + c.id);

      if (codeEl) {
        // Format as "123 456" for 6 digits or spaced for 8 digits
        const formatted = c.code.length === 6 ? c.code.replace(/(\d{3})(\d{3})/, '$1 $2') : c.code;
        codeEl.textContent = formatted;
        codeEl.dataset.raw = c.code;
      }

      if (timerText) {
        timerText.textContent = c.seconds_remaining;
      }

      if (timerProgress) {
        const fraction = c.seconds_remaining / (c.period || 30);
        const offset = CIRCLE_CIRCUMFERENCE * (1 - fraction);
        timerProgress.style.strokeDashoffset = offset;
      }

      if (timerContainer) {
        timerContainer.classList.remove('timer-warning', 'timer-danger');
        if (c.seconds_remaining <= 5) {
          timerContainer.classList.add('timer-danger');
        } else if (c.seconds_remaining <= 10) {
          timerContainer.classList.add('timer-warning');
        }
      }
    }
  } catch (err) {
    // Network hiccup - ignore and retry next second
  }
}

// Live Category & Search Filter
let currentCategoryFilter = 'all';

function filterCards() {
  const searchInput = document.getElementById('search-input');
  const query = (searchInput ? searchInput.value : '').toLowerCase().trim();
  const cards = document.querySelectorAll('.account-card');
  let visibleCount = 0;

  cards.forEach(card => {
    const issuer = (card.dataset.issuer || '').toLowerCase();
    const name = (card.dataset.name || '').toLowerCase();
    const category = (card.dataset.category || 'Personal').toLowerCase();

    const matchesSearch = !query || issuer.includes(query) || name.includes(query) || category.includes(query);
    const matchesCategory = (currentCategoryFilter === 'all') || (category === currentCategoryFilter.toLowerCase());

    const isVisible = matchesSearch && matchesCategory;
    card.style.display = isVisible ? 'block' : 'none';
    if (isVisible) visibleCount++;
  });

  const noResults = document.getElementById('no-search-results');
  if (noResults) {
    noResults.style.display = (visibleCount === 0 && cards.length > 0) ? 'block' : 'none';
  }
}

function setupSearch() {
  const searchInput = document.getElementById('search-input');
  if (!searchInput) return;
  searchInput.addEventListener('input', filterCards);
}

// Filter Pills Strip
function setupFilterPills() {
  const pills = document.querySelectorAll('.pill-chip[data-filter]');
  pills.forEach(pill => {
    pill.addEventListener('click', () => {
      pills.forEach(p => p.classList.remove('active'));
      pill.classList.add('active');
      currentCategoryFilter = pill.dataset.filter || 'all';
      filterCards();
    });
  });
}

// Tab switcher for Add Account
function setupTabs() {
  const tabs = document.querySelectorAll('.tab-btn');
  tabs.forEach(tab => {
    tab.addEventListener('click', () => {
      tabs.forEach(t => t.classList.remove('active'));
      tab.classList.add('active');

      const target = tab.dataset.tab;
      document.querySelectorAll('.tab-pane').forEach(pane => {
        pane.style.display = (pane.id === target) ? 'block' : 'none';
      });
    });
  });
}

// QR Code File Drop / Scanner support
function setupQRScanner() {
  const dropzone = document.getElementById('qr-dropzone');
  const fileInput = document.getElementById('qr-file-input');
  if (!dropzone || !fileInput) return;

  dropzone.addEventListener('click', () => fileInput.click());

  dropzone.addEventListener('dragover', (e) => {
    e.preventDefault();
    dropzone.style.borderColor = '#059669';
    dropzone.style.background = '#dcfce7';
  });

  dropzone.addEventListener('dragleave', () => {
    dropzone.style.borderColor = '#86efac';
    dropzone.style.background = '#f0fdf4';
  });

  dropzone.addEventListener('drop', (e) => {
    e.preventDefault();
    dropzone.style.borderColor = '#86efac';
    dropzone.style.background = '#f0fdf4';
    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      handleQRFile(e.dataTransfer.files[0]);
    }
  });

  fileInput.addEventListener('change', (e) => {
    if (e.target.files && e.target.files[0]) {
      handleQRFile(e.target.files[0]);
    }
  });
}

async function handleQRFile(file) {
  if (!file.type.startsWith('image/')) {
    showToast('Please upload an image screenshot (PNG/JPG).');
    return;
  }

  // Check if browser has native BarcodeDetector
  if ('BarcodeDetector' in window) {
    try {
      const barcodeDetector = new BarcodeDetector({ formats: ['qr_code'] });
      const imageBitmap = await createImageBitmap(file);
      const barcodes = await barcodeDetector.detect(imageBitmap);
      if (barcodes.length > 0) {
        populateFromURI(barcodes[0].rawValue);
        return;
      }
    } catch (e) {
      console.warn('BarcodeDetector error:', e);
    }
  }

  showToast('Image uploaded! Paste secret or URI if auto-detection is not available.');
}

function populateFromURI(uri) {
  const uriInput = document.getElementById('input-uri-secret');
  if (uriInput) {
    uriInput.value = uri;
    showToast('QR Code parsed successfully!');
    const manualTab = document.querySelector('[data-tab="tab-manual"]');
    if (manualTab) manualTab.click();
  }
}

// Auto-fill confirmation code live preview when adding & Category Sync
function setupAddHelper() {
  const secretInput = document.getElementById('otpauth_or_secret') || document.getElementById('input-uri-secret');
  if (secretInput) {
    secretInput.addEventListener('input', () => {
      const val = secretInput.value.trim();
      if (val.startsWith('otpauth://')) {
        try {
          const u = new URL(val);
          const issuer = u.searchParams.get('issuer');
          const issuerInput = document.getElementById('issuer') || document.getElementById('input-issuer');
          if (issuer && issuerInput && !issuerInput.value) {
            issuerInput.value = issuer;
          }
        } catch (e) { }
      }
    });
  }

  const catInput = document.getElementById('input-category');
  const hiddenInput = document.getElementById('hidden-category');
  const selectCat = document.getElementById('select-category');
  const addForm = document.getElementById('add-token-form');

  if (catInput && hiddenInput) {
    catInput.addEventListener('input', () => {
      hiddenInput.value = catInput.value;
    });
  }

  if (addForm && selectCat && hiddenInput && catInput) {
    addForm.addEventListener('submit', () => {
      if (selectCat.value === 'custom') {
        hiddenInput.value = catInput.value.trim() || 'General';
      } else {
        hiddenInput.value = selectCat.value;
      }
    });
  }

  if (selectCat) {
    handleCategorySelect(selectCat.value);
  }
}

// Category helper for Add Token page
function handleCategorySelect(val) {
  const catInput = document.getElementById('input-category');
  const hiddenInput = document.getElementById('hidden-category');
  if (!catInput) return;

  if (val === 'custom') {
    catInput.disabled = false;
    catInput.value = '';
    catInput.placeholder = 'Enter custom category name...';
    catInput.focus();
    if (hiddenInput) hiddenInput.value = '';
  } else {
    catInput.disabled = true;
    catInput.value = val;
    catInput.placeholder = 'Custom category name...';
    if (hiddenInput) hiddenInput.value = val;
  }
}

// Custom Accessible & Beautiful Select Dropdown Component
function initCustomSelects() {
  const selects = document.querySelectorAll('select:not([data-custom-select-ready])');

  selects.forEach(select => {
    select.setAttribute('data-custom-select-ready', 'true');
    select.style.display = 'none';

    // Wrapper
    const wrapper = document.createElement('div');
    wrapper.className = 'custom-select-wrapper';

    // Copy width/flex inline style from select if present
    const originalWidth = select.style.width;
    const originalFlex = select.style.flexShrink;
    if (originalWidth) wrapper.style.width = originalWidth;
    if (originalFlex) wrapper.style.flexShrink = originalFlex;

    // Trigger
    const trigger = document.createElement('div');
    trigger.className = 'custom-select-trigger';
    trigger.setAttribute('tabindex', '0');
    trigger.setAttribute('role', 'combobox');
    trigger.setAttribute('aria-expanded', 'false');

    const selectedOption = select.options[select.selectedIndex] || select.options[0];
    const triggerText = document.createElement('span');
    triggerText.className = 'custom-select-text';
    triggerText.textContent = selectedOption ? selectedOption.textContent : 'Select...';

    const arrow = document.createElement('span');
    arrow.className = 'custom-select-arrow';
    arrow.innerHTML = `<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M6 8l4 4 4-4"/></svg>`;

    trigger.appendChild(triggerText);
    trigger.appendChild(arrow);
    wrapper.appendChild(trigger);

    // Dropdown List
    const dropdown = document.createElement('div');
    dropdown.className = 'custom-select-dropdown';
    dropdown.setAttribute('role', 'listbox');

    function buildOptions() {
      dropdown.innerHTML = '';
      Array.from(select.options).forEach((opt, idx) => {
        const item = document.createElement('div');
        item.className = 'custom-select-item';
        item.setAttribute('role', 'option');
        item.setAttribute('data-value', opt.value);
        item.textContent = opt.textContent;

        if (opt.selected || idx === select.selectedIndex) {
          item.classList.add('selected');
          const check = document.createElement('span');
          check.className = 'custom-select-checkmark';
          check.textContent = '✓';
          item.appendChild(check);
        }

        item.addEventListener('click', (e) => {
          e.stopPropagation();
          select.value = opt.value;
          select.selectedIndex = idx;
          triggerText.textContent = opt.textContent;

          wrapper.classList.remove('open');
          trigger.setAttribute('aria-expanded', 'false');

          // Refresh selected styling in dropdown list
          dropdown.querySelectorAll('.custom-select-item').forEach(el => {
            el.classList.remove('selected');
            const c = el.querySelector('.custom-select-checkmark');
            if (c) c.remove();
          });
          item.classList.add('selected');
          const check = document.createElement('span');
          check.className = 'custom-select-checkmark';
          check.textContent = '✓';
          item.appendChild(check);

          // Dispatch native change & input events
          select.dispatchEvent(new Event('change', { bubbles: true }));
          select.dispatchEvent(new Event('input', { bubbles: true }));
          trigger.focus();
        });

        dropdown.appendChild(item);
      });
    }

    buildOptions();
    wrapper.appendChild(dropdown);

    // Toggle dropdown on click
    trigger.addEventListener('click', (e) => {
      e.stopPropagation();
      const isOpen = wrapper.classList.contains('open');
      closeAllCustomSelects();
      if (!isOpen) {
        wrapper.classList.add('open');
        trigger.setAttribute('aria-expanded', 'true');
      }
    });

    // Keyboard navigation
    trigger.addEventListener('keydown', (e) => {
      if (e.key === 'Enter' || e.key === ' ' || e.key === 'ArrowDown' || e.key === 'ArrowUp') {
        e.preventDefault();
        if (!wrapper.classList.contains('open')) {
          closeAllCustomSelects();
          wrapper.classList.add('open');
          trigger.setAttribute('aria-expanded', 'true');
        } else {
          if (e.key === 'ArrowDown') {
            const next = (select.selectedIndex + 1) % select.options.length;
            select.selectedIndex = next;
            select.value = select.options[next].value;
            triggerText.textContent = select.options[next].textContent;
            buildOptions();
            select.dispatchEvent(new Event('change', { bubbles: true }));
          } else if (e.key === 'ArrowUp') {
            const prev = (select.selectedIndex - 1 + select.options.length) % select.options.length;
            select.selectedIndex = prev;
            select.value = select.options[prev].value;
            triggerText.textContent = select.options[prev].textContent;
            buildOptions();
            select.dispatchEvent(new Event('change', { bubbles: true }));
          } else if (e.key === 'Enter' || e.key === 'Escape') {
            wrapper.classList.remove('open');
            trigger.setAttribute('aria-expanded', 'false');
          }
        }
      } else if (e.key === 'Escape') {
        wrapper.classList.remove('open');
        trigger.setAttribute('aria-expanded', 'false');
      }
    });

    // Insert wrapper into DOM
    select.parentNode.insertBefore(wrapper, select);
    wrapper.appendChild(select);

    // Sync when select value changes externally
    select.addEventListener('change', () => {
      const currentOpt = select.options[select.selectedIndex];
      if (currentOpt) {
        triggerText.textContent = currentOpt.textContent;
        buildOptions();
      }
    });
  });
}

function closeAllCustomSelects() {
  document.querySelectorAll('.custom-select-wrapper.open').forEach(w => {
    w.classList.remove('open');
    const t = w.querySelector('.custom-select-trigger');
    if (t) t.setAttribute('aria-expanded', 'false');
  });
}

// Global click outside to close dropdowns
document.addEventListener('click', () => {
  closeAllCustomSelects();
});

// Styled Confirmation Modal Controller
function openDeleteModal(id, issuer, accountName) {
  const modal = document.getElementById('delete-modal');
  const tokenNameEl = document.getElementById('delete-token-name');
  const tokenIdInput = document.getElementById('delete-token-id');
  if (!modal || !tokenNameEl || !tokenIdInput) return;

  const displayName = accountName ? `${issuer} (${accountName})` : issuer;
  tokenNameEl.textContent = displayName;
  tokenIdInput.value = id;
  modal.style.display = 'flex';
}

function closeDeleteModal() {
  const modal = document.getElementById('delete-modal');
  if (modal) modal.style.display = 'none';
}

function openDeleteUserModal(id, username) {
  const modal = document.getElementById('delete-user-modal');
  const userNameEl = document.getElementById('delete-user-name');
  const userIdInput = document.getElementById('delete-user-id');
  if (!modal || !userNameEl || !userIdInput) return;

  userNameEl.textContent = username;
  userIdInput.value = id;
  modal.style.display = 'flex';
}

function closeDeleteUserModal() {
  const modal = document.getElementById('delete-user-modal');
  if (modal) modal.style.display = 'none';
}

// Local Vector Service Brand Icons Registry
const SERVICE_ICONS = {
  'google': '/static/icons/google.svg',
  'gmail': '/static/icons/google.svg',
  'reddit': '/static/icons/reddit.svg',
  'github': '/static/icons/github.svg',
  'discord': '/static/icons/discord.svg',
  'aws': '/static/icons/aws.svg',
  'amazon': '/static/icons/aws.svg',
  'microsoft': '/static/icons/microsoft.svg',
  'azure': '/static/icons/microsoft.svg',
  'outlook': '/static/icons/microsoft.svg',
  'office365': '/static/icons/microsoft.svg',
  'apple': '/static/icons/apple.svg',
  'icloud': '/static/icons/apple.svg',
  'twitter': '/static/icons/x.svg',
  'x.com': '/static/icons/x.svg',
  'gitlab': '/static/icons/gitlab.svg',
  'bitbucket': '/static/icons/bitbucket.svg',
  'binance': '/static/icons/binance.svg',
  'coinbase': '/static/icons/coinbase.svg',
  'paypal': '/static/icons/paypal.svg',
  'proton': '/static/icons/proton.svg',
  'protonmail': '/static/icons/proton.svg',
  'slack': '/static/icons/slack.svg',
  'dropbox': '/static/icons/dropbox.svg',
  'spotify': '/static/icons/spotify.svg',
  'steam': '/static/icons/steam.svg',
  'epic': '/static/icons/epicgames.svg',
  'epicgames': '/static/icons/epicgames.svg',
  'twitch': '/static/icons/twitch.svg',
  'cloudflare': '/static/icons/cloudflare.svg',
  'digitalocean': '/static/icons/digitalocean.svg',
  'stripe': '/static/icons/stripe.svg',
  'openai': '/static/icons/openai.svg',
  'chatgpt': '/static/icons/openai.svg',
  'facebook': '/static/icons/facebook.svg',
  'meta': '/static/icons/facebook.svg',
  'instagram': '/static/icons/instagram.svg',
  'linkedin': '/static/icons/linkedin.svg'
};

const PASTEL_PALETTE = [
  { bg: '#dcfce7', text: '#15803d', border: '#bbf7d0' }, // Emerald
  { bg: '#e0e7ff', text: '#4338ca', border: '#c7d2fe' }, // Indigo
  { bg: '#fef3c7', text: '#b45309', border: '#fde68a' }, // Amber
  { bg: '#f3e8ff', text: '#7e22ce', border: '#e9d5ff' }, // Purple
  { bg: '#e0f2fe', text: '#0369a1', border: '#bae6fd' }, // Sky
  { bg: '#ffe4e6', text: '#be123c', border: '#fecdd3' }, // Rose
  { bg: '#f1f5f9', text: '#334155', border: '#cbd5e1' }  // Slate
];

function getMonogramColor(str) {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    hash = str.charCodeAt(i) + ((hash << 5) - hash);
  }
  const index = Math.abs(hash) % PASTEL_PALETTE.length;
  return PASTEL_PALETTE[index];
}

function renderServiceIcons() {
  document.querySelectorAll('.service-icon[data-issuer]').forEach(el => {
    const issuer = (el.getAttribute('data-issuer') || '').trim();
    const lower = issuer.toLowerCase();

    let matchIcon = null;
    for (const [key, iconPath] of Object.entries(SERVICE_ICONS)) {
      if (lower.includes(key)) {
        matchIcon = iconPath;
        break;
      }
    }

    if (matchIcon) {
      el.innerHTML = `<img src="${matchIcon}" width="24" height="24" alt="${issuer}" style="object-fit: contain;">`;
      el.style.backgroundColor = '#ffffff';
      el.style.border = '1.5px solid var(--card-border)';
    } else {
      const initials = (issuer.length >= 2 ? issuer.substring(0, 2) : (issuer.substring(0, 1) || 'OT')).toUpperCase();
      const colorScheme = getMonogramColor(issuer || 'Ottie');
      el.innerHTML = `<span>${initials}</span>`;
      el.style.backgroundColor = colorScheme.bg;
      el.style.color = colorScheme.text;
      el.style.border = `1.5px solid ${colorScheme.border}`;
      el.style.fontWeight = '800';
      el.style.fontSize = '12px';
      el.style.letterSpacing = '0.5px';
    }
  });
}

// Global modal close on Escape or backdrop click
document.addEventListener('keydown', (e) => {
  if (e.key === 'Escape') {
    closeDeleteModal();
    closeDeleteUserModal();
  }
});

// Initialize on DOM Ready
document.addEventListener('DOMContentLoaded', () => {
  setupSearch();
  setupFilterPills();
  setupTabs();
  setupQRScanner();
  setupAddHelper();
  initCustomSelects();
  renderServiceIcons();

  // If on dashboard, start live ticker
  if (document.getElementById('accounts-list')) {
    refreshCodes();
    setInterval(refreshCodes, 1000);
  }
});

