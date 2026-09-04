import { appBasePath } from './app-paths';

function configuredDashboardURL(): string | undefined {
  return (window as Window & { __DASHBOARD_URL__?: string }).__DASHBOARD_URL__;
}

export function coresDashboardURL(): string {
  if (appBasePath) return `${window.location.origin}/`;
  if (configuredDashboardURL()) return configuredDashboardURL()!;
  const { hostname, port, protocol } = window.location;
  if (port === '8081') return `${protocol}//${hostname}:8080`;
  if (/^(rent|rental)\./.test(hostname)) return `${protocol}//${hostname.replace(/^(rent|rental)\./, 'cores.')}`;
  return `${protocol}//${hostname}${port ? `:${port}` : ''}`;
}

export function centralLoginURL(): string {
  const dashboard = new URL(coresDashboardURL(), window.location.origin);
  const login = new URL('/login', dashboard);
  const localLoginPath = `${appBasePath}/login`;
  const current = window.location.pathname === localLoginPath
    ? `${appBasePath || ''}/`
    : `${window.location.pathname}${window.location.search}${window.location.hash}`;
  login.searchParams.set('redirect', dashboard.origin === window.location.origin ? current : new URL(current, window.location.origin).toString());
  return login.toString();
}
