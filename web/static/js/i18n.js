// RentalCore i18n System
// Uses i18next for translation management

(function() {
    'use strict';

    // Initialize i18next
    let i18nextInstance;

    // Get the suite-wide language or fall back to the browser preference.
    const getStoredLanguage = () => {
        const stored = localStorage.getItem('cores_language');
        if (stored === 'de' || stored === 'en') return stored;
        return navigator.language.toLowerCase().startsWith('en') ? 'en' : 'de';
    };

    // Save language preference
    const saveLanguage = (lang) => {
        localStorage.setItem('cores_language', lang);
    };

    // Load translations from JSON files
    const loadTranslations = async () => {
        try {
            const [deResponse, enResponse] = await Promise.all([
                fetch('/static/locales/de.json'),
                fetch('/static/locales/en.json')
            ]);

            const de = await deResponse.json();
            const en = await enResponse.json();

            return { de, en };
        } catch (error) {
            console.error('Failed to load translations:', error);
            return null;
        }
    };

    // Initialize i18next from CDN
    const initI18next = async () => {
        // Load i18next from CDN if not already loaded
        if (typeof i18next === 'undefined') {
            await loadScript('https://cdn.jsdelivr.net/npm/i18next@23.7.6/i18next.min.js');
        }

        const translations = await loadTranslations();
        if (!translations) {
            console.error('Could not initialize i18n - translations not loaded');
            return;
        }

        i18nextInstance = i18next.createInstance();
        await i18nextInstance.init({
            lng: getStoredLanguage(),
            fallbackLng: 'de',
            resources: {
                de: { translation: translations.de },
                en: { translation: translations.en }
            },
            interpolation: {
                escapeValue: false
            }
        });

        return i18nextInstance;
    };

    // Helper to load external scripts
    const loadScript = (src) => {
        return new Promise((resolve, reject) => {
            const script = document.createElement('script');
            script.src = src;
            script.onload = resolve;
            script.onerror = reject;
            document.head.appendChild(script);
        });
    };

    // Translate all elements with data-i18n attribute
    const translateElements = () => {
        if (!i18nextInstance) return;

        // Translate all elements with data-i18n attribute
        document.querySelectorAll('[data-i18n]').forEach(element => {
            const key = element.getAttribute('data-i18n');
            const translation = i18nextInstance.t(key);

            // Check if we should translate innerHTML or a specific attribute
            const attr = element.getAttribute('data-i18n-attr');
            if (attr) {
                element.setAttribute(attr, translation);
            } else {
                element.textContent = translation;
            }
        });
    };

    // Change language
    window.changeLanguage = async (lang) => {
        if (!i18nextInstance) return;

        await i18nextInstance.changeLanguage(lang);
        saveLanguage(lang);
        translateElements();

        // Update language switcher UI
        updateLanguageSwitcher(lang);

        // Dispatch custom event for other scripts to react to language change
        document.dispatchEvent(new CustomEvent('languageChanged', { detail: { language: lang } }));
        window.dispatchEvent(new CustomEvent('cores:languagechange', { detail: { language: lang } }));
        window.location.reload();
    };

    // Update language switcher UI
    const updateLanguageSwitcher = (lang) => {
        const switcher = document.getElementById('languageSwitcher');
        if (!switcher) return;

        const flag = lang === 'de' ? '🇩🇪' : '🇺🇸';
        const flagElement = switcher.querySelector('.current-flag');
        if (flagElement) {
            flagElement.textContent = flag;
        }

        // Update active state in dropdown
        switcher.querySelectorAll('.lang-option').forEach(option => {
            const optionLang = option.getAttribute('data-lang');
            if (optionLang === lang) {
                option.classList.add('active');
            } else {
                option.classList.remove('active');
            }
        });
    };

    // Create and inject language switcher
    const createLanguageSwitcher = () => {
        const currentLang = getStoredLanguage();
        const currentFlag = currentLang === 'de' ? '🇩🇪' : '🇺🇸';

        const switcherHTML = `
            <div class="rc-language-switcher" id="languageSwitcher">
                <button class="rc-lang-toggle" aria-label="Change Language">
                    <i class="bi bi-translate"></i>
                    <span class="current-flag">${currentFlag}</span>
                </button>
                <div class="rc-lang-dropdown">
                    <button class="lang-option ${currentLang === 'de' ? 'active' : ''}" data-lang="de" onclick="changeLanguage('de')">
                        <span class="lang-flag">🇩🇪</span>
                        <span class="lang-name">Deutsch</span>
                        ${currentLang === 'de' ? '<i class="bi bi-check-lg"></i>' : ''}
                    </button>
                    <button class="lang-option ${currentLang === 'en' ? 'active' : ''}" data-lang="en" onclick="changeLanguage('en')">
                        <span class="lang-flag">🇺🇸</span>
                        <span class="lang-name">English</span>
                        ${currentLang === 'en' ? '<i class="bi bi-check-lg"></i>' : ''}
                    </button>
                </div>
            </div>
        `;

        // Inject into header
        const headerRight = document.querySelector('.rc-header-right');
        if (headerRight) {
            headerRight.insertAdjacentHTML('afterbegin', switcherHTML);
        }
    };

    // Inject language switcher CSS
    const injectLanguageSwitcherCSS = () => {
        const style = document.createElement('style');
        style.textContent = `
            .rc-language-switcher {
                position: relative;
                margin-right: 1rem;
            }

            .rc-lang-toggle {
                display: flex;
                align-items: center;
                gap: 0.5rem;
                padding: 0.5rem 0.75rem;
                background: rgba(255, 255, 255, 0.05);
                border: none;
                border-radius: 0.5rem;
                color: rgba(255, 255, 255, 0.7);
                cursor: pointer;
                transition: all 0.2s;
            }

            .rc-lang-toggle:hover {
                background: rgba(255, 255, 255, 0.1);
                color: white;
            }

            .rc-lang-toggle i {
                font-size: 1.1rem;
            }

            .current-flag {
                font-size: 1.2rem;
                line-height: 1;
            }

            .rc-lang-dropdown {
                position: absolute;
                right: 0;
                top: calc(100% + 0.5rem);
                min-width: 12rem;
                background: var(--rc-card-bg, #1a1d29);
                border: 1px solid rgba(255, 255, 255, 0.1);
                border-radius: 0.5rem;
                box-shadow: 0 4px 6px rgba(0, 0, 0, 0.3);
                opacity: 0;
                visibility: hidden;
                transform: translateY(-10px);
                transition: all 0.2s;
                z-index: 1000;
            }

            .rc-language-switcher:hover .rc-lang-dropdown {
                opacity: 1;
                visibility: visible;
                transform: translateY(0);
            }

            .lang-option {
                display: flex;
                align-items: center;
                gap: 0.75rem;
                width: 100%;
                padding: 0.75rem 1rem;
                background: transparent;
                border: none;
                color: rgba(255, 255, 255, 0.7);
                cursor: pointer;
                transition: all 0.2s;
                text-align: left;
            }

            .lang-option:hover {
                background: rgba(255, 255, 255, 0.05);
                color: white;
            }

            .lang-option.active {
                background: rgba(239, 68, 68, 0.2);
                color: #ef4444;
            }

            .lang-option.active .bi-check-lg {
                margin-left: auto;
            }

            .lang-flag {
                font-size: 1.5rem;
                line-height: 1;
            }

            .lang-name {
                font-weight: 500;
            }

            @media (max-width: 768px) {
                .rc-language-switcher {
                    margin-right: 0;
                }

                .rc-lang-toggle {
                    padding: 0.4rem 0.6rem;
                }

                .rc-lang-dropdown {
                    min-width: 10rem;
                }
            }
        `;
        document.head.appendChild(style);
    };

    // Expose t function globally for use in templates
    window.t = (key, options) => {
        return i18nextInstance ? i18nextInstance.t(key, options) : key;
    };

    // Initialize on DOM ready
    const init = async () => {
        await initI18next();
        injectLanguageSwitcherCSS();
        createLanguageSwitcher();
        translateElements();

        // Auto-translate on new content (for dynamic pages)
        const observer = new MutationObserver((mutations) => {
            mutations.forEach((mutation) => {
                mutation.addedNodes.forEach((node) => {
                    if (node.nodeType === 1) { // Element node
                        const elements = node.querySelectorAll('[data-i18n]');
                        elements.forEach(element => {
                            const key = element.getAttribute('data-i18n');
                            const translation = i18nextInstance.t(key);
                            element.textContent = translation;
                        });
                    }
                });
            });
        });

        observer.observe(document.body, {
            childList: true,
            subtree: true
        });
    };

    // Start initialization when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();
