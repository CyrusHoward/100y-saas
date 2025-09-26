// Progressive Enhancement for 100y-saas
// Works with and without JavaScript

// State management
const state = {
  isOnline: navigator.onLine,
  isLoading: false,
  currentTenant: null,
  user: null
};

// Initialize application
document.addEventListener('DOMContentLoaded', function() {
  initializeApp();
});

async function initializeApp() {
  // Register service worker for offline support
  if ('serviceWorker' in navigator) {
    try {
      const registration = await navigator.serviceWorker.register('/web/sw.js');
      console.log('Service Worker registered:', registration);
      
      // Handle updates
      registration.addEventListener('updatefound', () => {
        const newWorker = registration.installing;
        newWorker.addEventListener('statechange', () => {
          if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
            showUpdateAvailable();
          }
        });
      });
    } catch (error) {
      console.log('Service Worker registration failed:', error);
    }
  }
  
  // Handle online/offline events
  window.addEventListener('online', () => {
    state.isOnline = true;
    updateConnectionStatus();
  });
  
  window.addEventListener('offline', () => {
    state.isOnline = false;
    updateConnectionStatus();
  });
  
  // Initialize UI
  setupEventListeners();
  loadInitialData();
  updateConnectionStatus();
}

function setupEventListeners() {
  // Enhanced form submission with progressive enhancement
  const form = document.getElementById('f');
  if (form) {
    form.addEventListener('submit', handleFormSubmit);
  }
  
  // Add loading states to buttons
  const buttons = document.querySelectorAll('button');
  buttons.forEach(button => {
    button.addEventListener('click', (e) => {
      if (e.target.type === 'submit') {
        setLoadingState(e.target, true);
      }
    });
  });
}

async function handleFormSubmit(e) {
  e.preventDefault();
  
  const title = document.getElementById('title').value.trim();
  const note = document.getElementById('note').value.trim();
  
  if (!title) {
    showError('Title is required');
    return;
  }
  
  const submitBtn = e.target.querySelector('button[type="submit"]') || e.target.querySelector('button');
  setLoadingState(submitBtn, true);
  
  try {
    const response = await fetch('/api/items', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ title, note })
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ error: 'Network error' }));
      throw new Error(errorData.error || `HTTP ${response.status}`);
    }
    
    const result = await response.json();
    
    // Clear form
    document.getElementById('title').value = '';
    document.getElementById('note').value = '';
    
    // Reload items
    await loadItems();
    
    showSuccess('Item added successfully');
    
  } catch (error) {
    console.error('Failed to add item:', error);
    
    if (!state.isOnline) {
      showError('You are offline. Please try again when connected.');
    } else {
      showError(`Failed to add item: ${error.message}`);
    }
  } finally {
    setLoadingState(submitBtn, false);
  }
}

async function loadInitialData() {
  await loadItems();
}

async function loadItems() {
  const ul = document.getElementById('list');
  if (!ul) return;
  
  try {
    setLoadingState(ul, true);
    
    const response = await fetch('/api/items');
    
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }
    
    const items = await response.json();
    
    // Clear existing items
    ul.innerHTML = '';
    
    if (items.length === 0) {
      ul.innerHTML = '<li class="no-items">No items yet. Add your first item above!</li>';
      return;
    }
    
    // Render items
    items.forEach(item => {
      const li = document.createElement('li');
      li.className = 'item';
      li.innerHTML = `
        <div class="item-content">
          <strong>${escapeHtml(item.title)}</strong>
          ${item.note ? `<p class="item-note">${escapeHtml(item.note)}</p>` : ''}
        </div>
        <div class="item-meta">
          <small>#${item.id}</small>
        </div>
      `;
      ul.appendChild(li);
    });
    
  } catch (error) {
    console.error('Failed to load items:', error);
    
    ul.innerHTML = `
      <li class="error-item">
        <strong>Failed to load items</strong>
        <p>${state.isOnline ? error.message : 'You are offline'}</p>
        <button onclick="loadItems()" class="retry-btn">Retry</button>
      </li>
    `;
  } finally {
    setLoadingState(ul, false);
  }
}

// Utility functions
function setLoadingState(element, loading) {
  if (!element) return;
  
  if (loading) {
    element.classList.add('loading');
    if (element.tagName === 'BUTTON') {
      element.disabled = true;
      element.dataset.originalText = element.textContent;
      element.textContent = 'Loading...';
    } else if (element.tagName === 'UL') {
      element.innerHTML = '<li class="loading-item">Loading items...</li>';
    }
  } else {
    element.classList.remove('loading');
    if (element.tagName === 'BUTTON') {
      element.disabled = false;
      if (element.dataset.originalText) {
        element.textContent = element.dataset.originalText;
        delete element.dataset.originalText;
      }
    }
  }
}

function updateConnectionStatus() {
  const statusEl = document.getElementById('connection-status') || createConnectionStatus();
  
  if (state.isOnline) {
    statusEl.className = 'connection-status online';
    statusEl.textContent = 'Online';
    statusEl.style.display = 'none';
  } else {
    statusEl.className = 'connection-status offline';
    statusEl.textContent = '📱 Offline Mode';
    statusEl.style.display = 'block';
  }
}

function createConnectionStatus() {
  const statusEl = document.createElement('div');
  statusEl.id = 'connection-status';
  document.body.insertBefore(statusEl, document.body.firstChild);
  return statusEl;
}

function showError(message) {
  showNotification(message, 'error');
}

function showSuccess(message) {
  showNotification(message, 'success');
}

function showNotification(message, type = 'info') {
  // Remove existing notifications
  const existing = document.querySelectorAll('.notification');
  existing.forEach(el => el.remove());
  
  const notification = document.createElement('div');
  notification.className = `notification ${type}`;
  notification.textContent = message;
  
  // Add close button
  const closeBtn = document.createElement('button');
  closeBtn.className = 'notification-close';
  closeBtn.textContent = '×';
  closeBtn.onclick = () => notification.remove();
  notification.appendChild(closeBtn);
  
  document.body.appendChild(notification);
  
  // Auto-remove after 5 seconds
  setTimeout(() => {
    if (notification.parentNode) {
      notification.remove();
    }
  }, 5000);
}

function showUpdateAvailable() {
  const updateBanner = document.createElement('div');
  updateBanner.className = 'update-banner';
  updateBanner.innerHTML = `
    <p>A new version is available!</p>
    <button onclick="reloadForUpdate()">Update Now</button>
    <button onclick="dismissUpdate(this)">Later</button>
  `;
  document.body.insertBefore(updateBanner, document.body.firstChild);
}

function reloadForUpdate() {
  if (navigator.serviceWorker.controller) {
    navigator.serviceWorker.controller.postMessage({ type: 'SKIP_WAITING' });
  }
  window.location.reload();
}

function dismissUpdate(button) {
  button.closest('.update-banner').remove();
}

function escapeHtml(text) {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

// Initialize on load (fallback for non-modern browsers)
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', initializeApp);
} else {
  initializeApp();
}
