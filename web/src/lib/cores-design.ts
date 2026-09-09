export type SuiteIdentity = object | null | undefined;

function stringValue(identity: SuiteIdentity, keys: string[]) {
  if (!identity) return '';
  const values = identity as Record<string, unknown>;
  for (const key of keys) {
    const value = values[key];
    if (typeof value === 'string' && value.trim()) return value.trim();
  }
  return '';
}

export function suiteGreetingName(identity: SuiteIdentity) {
  const firstName = stringValue(identity, ['firstName', 'first_name', 'FirstName']);
  if (firstName) return firstName;

  const displayName = stringValue(identity, ['displayName', 'display_name', 'DisplayName']);
  if (displayName) return displayName.split(/\s+/)[0];

  return stringValue(identity, ['username', 'Username']);
}

export function suiteGreeting(identity?: SuiteIdentity, now = new Date()) {
  const hour = now.getHours();
  const salutation = hour < 11 ? 'Guten Morgen' : hour < 18 ? 'Guten Tag' : 'Guten Abend';
  const name = suiteGreetingName(identity);
  return `${salutation}${name ? `, ${name}` : ''}.`;
}

export function suiteDateLabel(now = new Date()) {
  return new Intl.DateTimeFormat('de-DE', {
    weekday: 'long',
    day: '2-digit',
    month: 'long',
    year: 'numeric',
  }).format(now);
}

export type SuiteCoreKey = 'rental' | 'warehouse' | 'planner' | 'procurement';

export type SuiteNavigationConfig = {
  routingMode: 'paths' | 'subdomains';
  rentalUrl: string;
  warehouseUrl: string;
  plannerUrl: string;
  procurementUrl: string;
};

export const suiteCoreLabels: Record<SuiteCoreKey, string> = {
  rental: 'RentalCore',
  warehouse: 'WarehouseCore',
  planner: 'PlannerCore',
  procurement: 'ProcurementCore',
};

const suiteMounts: Record<SuiteCoreKey, string> = {
  rental: '/rentalcore/',
  warehouse: '/warehousecore/',
  planner: '/plannercore/',
  procurement: '/procurementcore/',
};

function inferredServiceOrigin(dashboard: URL, prefix: string, port: string) {
  if (dashboard.hostname === 'localhost' || dashboard.hostname === '127.0.0.1') {
    return `${dashboard.protocol}//${dashboard.hostname}:${port}/`;
  }
  if (dashboard.hostname.startsWith('cores.')) {
    return `${dashboard.protocol}//${prefix}.${dashboard.hostname.slice('cores.'.length)}/`;
  }
  return dashboard.origin + suiteMounts[prefix === 'rent' ? 'rental' : prefix as SuiteCoreKey];
}

export function suiteNavigationFallback(dashboardURL: string, currentPath = window.location.pathname): SuiteNavigationConfig {
  const dashboard = new URL(dashboardURL || '/', window.location.origin);
  const pathMode = Object.values(suiteMounts).some((mount) => currentPath === mount.slice(0, -1) || currentPath.startsWith(mount));
  if (pathMode || dashboard.origin === window.location.origin) {
    return { routingMode: 'paths', rentalUrl: suiteMounts.rental, warehouseUrl: suiteMounts.warehouse, plannerUrl: suiteMounts.planner, procurementUrl: suiteMounts.procurement };
  }
  return {
    routingMode: 'subdomains',
    rentalUrl: inferredServiceOrigin(dashboard, 'rent', '8081'),
    warehouseUrl: inferredServiceOrigin(dashboard, 'warehouse', '8082'),
    plannerUrl: inferredServiceOrigin(dashboard, 'planner', '8083'),
    procurementUrl: inferredServiceOrigin(dashboard, 'procurement', '8084'),
  };
}

export async function loadSuiteNavigation(dashboardURL: string, signal?: AbortSignal): Promise<SuiteNavigationConfig> {
  const fallback = suiteNavigationFallback(dashboardURL);
  try {
    const endpoint = new URL('/api/v1/config', new URL(dashboardURL || '/', window.location.origin));
    const response = await fetch(endpoint, { signal });
    if (!response.ok) return fallback;
    const value = await response.json() as Partial<SuiteNavigationConfig>;
    if (!value.rentalUrl || !value.warehouseUrl || !value.plannerUrl || !value.procurementUrl) return fallback;
    return {
      routingMode: value.routingMode === 'subdomains' ? 'subdomains' : 'paths',
      rentalUrl: value.rentalUrl,
      warehouseUrl: value.warehouseUrl,
      plannerUrl: value.plannerUrl,
      procurementUrl: value.procurementUrl,
    };
  } catch {
    return fallback;
  }
}
