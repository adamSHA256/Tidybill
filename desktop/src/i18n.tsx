import { createContext, useContext, useState, useEffect, type ReactNode } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from './api/client'

import cs from './locales/cs.json'
import sk from './locales/sk.json'
import en from './locales/en.json'

type Lang = 'cs' | 'sk' | 'en'
type Messages = Record<string, string>

const locales: Record<Lang, Messages> = { cs, sk, en }

interface I18nContextValue {
  t: (key: string) => string
  lang: Lang
  setLang: (lang: Lang) => void
}

const I18nContext = createContext<I18nContextValue>({
  t: (key) => key,
  lang: 'cs',
  setLang: () => {},
})

export function I18nProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Lang>('cs')
  const queryClient = useQueryClient()

  const { data: settings } = useQuery({
    queryKey: ['settings'],
    queryFn: api.getSettings,
  })

  useEffect(() => {
    if (settings?.language && settings.language in locales) {
      setLangState(settings.language as Lang)
    }
  }, [settings?.language])

  const setLang = (newLang: Lang) => {
    setLangState(newLang)
    api.updateSettings({ language: newLang }).then(() => {
      queryClient.invalidateQueries({ queryKey: ['settings'] })
      queryClient.invalidateQueries({ queryKey: ['templates'] })
    })
  }

  const t = (key: string): string => {
    return locales[lang]?.[key] ?? locales.cs[key] ?? key
  }

  return (
    <I18nContext.Provider value={{ t, lang, setLang }}>
      {children}
    </I18nContext.Provider>
  )
}

export function useT() {
  return useContext(I18nContext)
}
