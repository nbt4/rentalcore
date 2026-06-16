// toast.ts — lightweight toast notification utility
// Displays non-blocking error/success/info toasts in the bottom-right corner.

type ToastType = 'error' | 'success' | 'info';

let toastId = 0;
let container: HTMLDivElement | null = null;

function getContainer(): HTMLDivElement {
  if (!container) {
    container = document.createElement('div');
    container.id = 'cores-toast-container';
    container.style.cssText = `
      position: fixed;
      bottom: 24px;
      right: 24px;
      z-index: 99999;
      display: flex;
      flex-direction: column-reverse;
      gap: 8px;
      pointer-events: none;
    `;
    document.body.appendChild(container);
  }
  return container;
}

const COLORS: Record<ToastType, { bg: string; border: string; text: string }> = {
  error:   { bg: '#fee2e2', border: '#f87171', text: '#991b1b' },
  success: { bg: '#dcfce7', border: '#4ade80', text: '#166534' },
  info:    { bg: '#dbeafe', border: '#60a5fa', text: '#1e40af' },
};

function showToast(message: string, type: ToastType): void {
  const id = ++toastId;
  const colors = COLORS[type];

  const el = document.createElement('div');
  el.style.cssText = `
    background: ${colors.bg};
    color: ${colors.text};
    border: 1px solid ${colors.border};
    border-radius: 8px;
    padding: 12px 20px;
    font-size: 14px;
    font-family: system-ui, sans-serif;
    box-shadow: 0 4px 12px rgba(0,0,0,0.12);
    pointer-events: auto;
    animation: cores-toast-in 0.3s ease-out;
    max-width: 400px;
    word-break: break-word;
  `;
  el.id = `toast-${id}`;
  el.textContent = message;

  // Inject keyframes once
  if (!document.getElementById('cores-toast-style')) {
    const style = document.createElement('style');
    style.id = 'cores-toast-style';
    style.textContent = `
      @keyframes cores-toast-in {
        from { opacity: 0; transform: translateY(12px); }
        to   { opacity: 1; transform: translateY(0); }
      }
      @keyframes cores-toast-out {
        from { opacity: 1; transform: translateY(0); }
        to   { opacity: 0; transform: translateY(12px); }
      }
    `;
    document.head.appendChild(style);
  }

  getContainer().appendChild(el);

  // Auto-dismiss after 5 seconds
  setTimeout(() => {
    el.style.animation = 'cores-toast-out 0.3s ease-in forwards';
    el.addEventListener('animationend', () => {
      el.remove();
      // Clean up container if empty
      const c = getContainer();
      if (c.children.length === 0 && container) {
        container.remove();
        container = null;
      }
    });
  }, 5000);

  // Click to dismiss early
  el.addEventListener('click', () => {
    el.style.animation = 'cores-toast-out 0.3s ease-in forwards';
    el.addEventListener('animationend', () => el.remove());
  });
}

/** Extract a human-readable string from any error-like value */
function errMsg(e: unknown): string {
  if (e instanceof Error) return e.message;
  if (typeof e === 'string') return e;
  try { return JSON.stringify(e); } catch { return String(e); }
}

export const toast = {
  error: (e: unknown) => showToast(errMsg(e), 'error'),
  success: (message: string) => showToast(message, 'success'),
  info: (message: string) => showToast(message, 'info'),
};
