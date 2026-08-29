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
