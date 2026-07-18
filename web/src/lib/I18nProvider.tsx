import { useEffect, useState, type ReactNode } from 'react'
import { I18nContext, translations, type Lang, type Translations } from './i18n'

export function I18nProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Lang>(() => {
    const saved = localStorage.getItem('p2b_lang')
    return saved === 'en' ? 'en' : 'vi'
  })

  useEffect(() => {
    document.documentElement.lang = lang
  }, [lang])

  const setLang = (nextLang: Lang) => {
    setLangState(nextLang)
    localStorage.setItem('p2b_lang', nextLang)
  }

  const t = <T extends keyof Translations>(section: T): Translations[T] => {
    return translations[lang][section]
  }

  return <I18nContext.Provider value={{ lang, setLang, t }}>{children}</I18nContext.Provider>
}
