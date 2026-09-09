import { useEffect, useMemo, useState } from 'react';
import { Blocks, ChevronDown, LayoutDashboard } from 'lucide-react';
import {
  loadSuiteNavigation,
  suiteCoreLabels,
  suiteNavigationFallback,
  type SuiteCoreKey,
  type SuiteNavigationConfig,
} from '../lib/cores-design';

const coreKeys: SuiteCoreKey[] = ['rental', 'warehouse', 'planner', 'procurement'];

export function SuiteCoreNavigation({ current, dashboardURL, compact = false }: { current?: SuiteCoreKey; dashboardURL: string; compact?: boolean }) {
  const [navigation, setNavigation] = useState<SuiteNavigationConfig>(() => suiteNavigationFallback(dashboardURL));
  const destinations = useMemo<Record<SuiteCoreKey, string>>(() => ({
    rental: navigation.rentalUrl, warehouse: navigation.warehouseUrl, planner: navigation.plannerUrl, procurement: navigation.procurementUrl,
  }), [navigation]);
  useEffect(() => {
    const controller = new AbortController();
    void loadSuiteNavigation(dashboardURL, controller.signal).then(setNavigation);
    return () => controller.abort();
  }, [dashboardURL]);
  const changeCore = (next: string) => {
    if (!next || next === current) return;
    window.location.assign(new URL(destinations[next as SuiteCoreKey], window.location.origin).toString());
  };
  return <div className="suite-core-navigation" data-compact={compact}>
    <label className="suite-core-switcher">
      <span className="suite-core-switcher-label">Core wechseln</span>
      <span className="suite-core-switcher-control">
        <Blocks className="suite-core-switcher-icon" aria-hidden="true" />
        <select className="suite-core-switcher-select" aria-label="Core wechseln" value={current || ''} onChange={(event) => changeCore(event.target.value)}>
          {!current && <option value="">Core auswählen</option>}
          {coreKeys.map((key) => <option key={key} value={key}>{suiteCoreLabels[key]}</option>)}
        </select>
        <ChevronDown className="suite-core-switcher-chevron" aria-hidden="true" />
      </span>
    </label>
    <a className="suite-core-dashboard-link" href={dashboardURL} aria-current={current ? undefined : 'page'} title={compact ? 'Cores Dashboard' : undefined}>
      <LayoutDashboard size={17} aria-hidden="true" /><span>Cores Dashboard</span>
    </a>
  </div>;
}
