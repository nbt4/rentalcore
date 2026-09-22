export type SuiteIdentity = object | null | undefined;

export type SuiteLanguage = 'de' | 'en';
export type SuiteTranslationTree = { [key: string]: string | SuiteTranslationTree };

const suiteLanguageStorageKey = 'cores_language';
const suiteLanguageEvent = 'cores:languagechange';
const supportedSuiteLanguages: SuiteLanguage[] = ['de', 'en'];
let suiteTranslations: Record<SuiteLanguage, Record<string, string>> = { de: {}, en: {} };
let suiteTranslationPatterns: Record<SuiteLanguage, Array<{ pattern: RegExp; target: string; placeholders: string[] }>> = { de: [], en: [] };
let suiteI18nObserver: MutationObserver | undefined;
const suiteTextState = new WeakMap<Text, { source: string; output: string }>();
const suiteAttributeState = new WeakMap<Element, Map<string, { source: string; output: string }>>();
const translatedAttributes = ['aria-label', 'placeholder', 'title'] as const;

function browserSuiteLanguage(): SuiteLanguage {
  if (typeof window === 'undefined') return 'de';
  const currentURL = new URL(window.location.href);
  const requested = currentURL.searchParams.get('lang');
  if (requested && supportedSuiteLanguages.includes(requested as SuiteLanguage)) {
    window.localStorage.setItem(suiteLanguageStorageKey, requested);
    currentURL.searchParams.delete('lang');
    window.history.replaceState(window.history.state, '', `${currentURL.pathname}${currentURL.search}${currentURL.hash}`);
    return requested as SuiteLanguage;
  }
  const stored = window.localStorage.getItem(suiteLanguageStorageKey);
  if (stored && supportedSuiteLanguages.includes(stored as SuiteLanguage)) return stored as SuiteLanguage;
  return window.navigator.language.toLowerCase().startsWith('en') ? 'en' : 'de';
}

export function suiteLanguage(): SuiteLanguage {
  return browserSuiteLanguage();
}

export function suiteLocale(language = suiteLanguage()) {
  return language === 'en' ? 'en-GB' : 'de-DE';
}

export function suiteLocalizedURL(target: string) {
  const url = new URL(target, window.location.origin);
  url.searchParams.set('lang', suiteLanguage());
  return url.toString();
}

function flattenSuiteTranslations(tree: SuiteTranslationTree, result: Record<string, string> = {}) {
  Object.values(tree).forEach((value) => {
    if (typeof value === 'string') result[value] = value;
    else flattenSuiteTranslations(value, result);
  });
  return result;
}

export function pairSuiteTranslations(german: SuiteTranslationTree, english: SuiteTranslationTree) {
  const de = flattenSuiteTranslations(german);
  const enValues = flattenSuiteTranslations(english);
  const germanEntries: Array<[string, string]> = [];
  const englishEntries: Array<[string, string]> = [];

  function visit(deTree: SuiteTranslationTree, enTree: SuiteTranslationTree) {
    Object.entries(deTree).forEach(([key, deValue]) => {
      const enValue = enTree[key];
      if (typeof deValue === 'string' && typeof enValue === 'string') {
        germanEntries.push([deValue, deValue]);
        germanEntries.push([enValue, deValue]);
        englishEntries.push([deValue, enValue]);
        englishEntries.push([enValue, enValue]);
      } else if (typeof deValue === 'object' && typeof enValue === 'object') {
        visit(deValue, enValue);
      }
    });
  }

  visit(german, english);
  return {
    de: { ...de, ...Object.fromEntries(germanEntries) },
    en: { ...enValues, ...Object.fromEntries(englishEntries) },
  };
}

function compileTranslationPatterns(translations: Record<string, string>) {
  return Object.entries(translations).flatMap(([source, target]) => {
    const placeholders: string[] = [];
    let cursor = 0;
    let expression = '';
    const placeholderPattern = /{{\s*([\w.-]+)\s*}}/g;
    let match = placeholderPattern.exec(source);
    while (match) {
      expression += source.slice(cursor, match.index).replace(/[.*+?^${}()|[\]\\]/g, '\\$&').replace(/\s+/g, '\\s+');
      expression += '(.+?)';
      placeholders.push(match[1]);
      cursor = match.index + match[0].length;
      match = placeholderPattern.exec(source);
    }
    if (!placeholders.length) return [];
    expression += source.slice(cursor).replace(/[.*+?^${}()|[\]\\]/g, '\\$&').replace(/\s+/g, '\\s+');
    return [{ pattern: new RegExp(`^${expression}$`, 's'), target, placeholders }];
  });
}

function translatePattern(value: string, language: SuiteLanguage) {
  for (const entry of suiteTranslationPatterns[language]) {
    const match = value.match(entry.pattern);
    if (!match) continue;
    const replacements = Object.fromEntries(entry.placeholders.map((placeholder, index) => [placeholder, match[index + 1]]));
    return entry.target.replace(/{{\s*([\w.-]+)\s*}}/g, (_, placeholder: string) => replacements[placeholder] ?? '');
  }
  return value;
}

function translateValue(value: string, language = suiteLanguage()) {
  const whitespace = value.match(/^(\s*)(.*?)(\s*)$/s);
  if (!whitespace) return value;
  const translated = suiteTranslations[language][whitespace[2]] ?? translatePattern(whitespace[2], language);
  return translated ? `${whitespace[1]}${translated}${whitespace[3]}` : value;
}

export function suiteTranslate(value: string, language = suiteLanguage()) {
  return translateValue(value, language);
}

function translateTextNode(node: Text) {
  const value = node.nodeValue ?? '';
  const previous = suiteTextState.get(node);
  const source = previous && value === previous.output ? previous.source : value;
  const output = translateValue(source);
  suiteTextState.set(node, { source, output });
  if (value !== output) node.nodeValue = output;
}

function translateElementAttributes(element: Element) {
  const states = suiteAttributeState.get(element) ?? new Map();
  translatedAttributes.forEach((attribute) => {
    const value = element.getAttribute(attribute);
    if (value === null) return;
    const previous = states.get(attribute);
    const source = previous && value === previous.output ? previous.source : value;
    const output = translateValue(source);
    states.set(attribute, { source, output });
    if (value !== output) element.setAttribute(attribute, output);
  });
  suiteAttributeState.set(element, states);
}

function translateSubtree(root: Node) {
  if (root instanceof Text) {
    translateTextNode(root);
    return;
  }
  if (!(root instanceof Element) || root.matches('script, style, code, pre, [data-suite-i18n-ignore]')) return;
  translateElementAttributes(root);
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_ELEMENT | NodeFilter.SHOW_TEXT, {
    acceptNode(node) {
      if (node instanceof Element && node.matches('script, style, code, pre, [data-suite-i18n-ignore]')) {
        return NodeFilter.FILTER_REJECT;
      }
      return NodeFilter.FILTER_ACCEPT;
    },
  });
  let node = walker.nextNode();
  while (node) {
    if (node instanceof Text) translateTextNode(node);
    else if (node instanceof Element) translateElementAttributes(node);
    node = walker.nextNode();
  }
}

function applySuiteLanguage() {
  if (typeof document === 'undefined') return;
  document.documentElement.lang = suiteLanguage();
  if (document.body) translateSubtree(document.body);
}

export function setSuiteLanguage(language: SuiteLanguage) {
  if (!supportedSuiteLanguages.includes(language)) return;
  window.localStorage.setItem(suiteLanguageStorageKey, language);
  applySuiteLanguage();
  window.dispatchEvent(new CustomEvent(suiteLanguageEvent, { detail: { language } }));
}

export function onSuiteLanguageChange(listener: (language: SuiteLanguage) => void) {
  const handler = (event: Event) => listener((event as CustomEvent<{ language: SuiteLanguage }>).detail.language);
  window.addEventListener(suiteLanguageEvent, handler);
  return () => window.removeEventListener(suiteLanguageEvent, handler);
}

export function initSuiteI18n(translations: Partial<Record<SuiteLanguage, Record<string, string>>>) {
  suiteTranslations = {
    de: { ...suiteTranslations.de, ...translations.de },
    en: { ...suiteTranslations.en, ...translations.en },
  };
  suiteTranslationPatterns = {
    de: compileTranslationPatterns(suiteTranslations.de),
    en: compileTranslationPatterns(suiteTranslations.en),
  };
  if (typeof document === 'undefined') return;
  applySuiteLanguage();
  suiteI18nObserver?.disconnect();
  suiteI18nObserver = new MutationObserver((mutations) => {
    mutations.forEach((mutation) => {
      if (mutation.type === 'characterData') translateSubtree(mutation.target);
      mutation.addedNodes.forEach(translateSubtree);
      if (mutation.type === 'attributes') translateElementAttributes(mutation.target as Element);
    });
  });
  suiteI18nObserver.observe(document.documentElement, {
    subtree: true,
    childList: true,
    characterData: true,
    attributes: true,
    attributeFilter: [...translatedAttributes],
  });
}

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
  const english = suiteLanguage() === 'en';
  const salutation = english
    ? hour < 12 ? 'Good morning' : hour < 18 ? 'Good afternoon' : 'Good evening'
    : hour < 11 ? 'Guten Morgen' : hour < 18 ? 'Guten Tag' : 'Guten Abend';
  const name = suiteGreetingName(identity);
  return `${salutation}${name ? `, ${name}` : ''}.`;
}

export function suiteDateLabel(now = new Date()) {
  return new Intl.DateTimeFormat(suiteLocale(), {
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
