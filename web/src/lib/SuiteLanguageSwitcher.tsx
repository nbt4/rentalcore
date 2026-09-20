import { useEffect, useState } from 'react';
import {
  onSuiteLanguageChange,
  setSuiteLanguage,
  suiteLanguage,
  type SuiteLanguage,
} from './cores-design';

type SuiteLanguageSwitcherProps = {
  compact?: boolean;
};

export function SuiteLanguageSwitcher({ compact = false }: SuiteLanguageSwitcherProps) {
  const [language, setLanguage] = useState<SuiteLanguage>(() => suiteLanguage());

  useEffect(() => onSuiteLanguageChange(setLanguage), []);

  return (
    <label className={`suite-language-control${compact ? ' is-compact' : ''}`}>
      <span className="suite-language-label">Sprache</span>
      <select
        className="suite-language-select"
        aria-label="Sprache auswählen"
        value={language}
        onChange={(event) => {
          const nextLanguage = event.target.value as SuiteLanguage;
          if (nextLanguage === language) return;
          setSuiteLanguage(nextLanguage);
          window.location.reload();
        }}
      >
        <option value="de">Deutsch</option>
        <option value="en">English</option>
      </select>
    </label>
  );
}
