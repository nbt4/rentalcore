export function subscribeDashboardRefresh(refresh: () => void) {
  const refreshVisible = () => {
    if (document.visibilityState === 'visible') refresh();
  };
  const interval = window.setInterval(refreshVisible, 60_000);
  window.addEventListener('focus', refreshVisible);
  document.addEventListener('visibilitychange', refreshVisible);

  return () => {
    window.clearInterval(interval);
    window.removeEventListener('focus', refreshVisible);
    document.removeEventListener('visibilitychange', refreshVisible);
  };
}
