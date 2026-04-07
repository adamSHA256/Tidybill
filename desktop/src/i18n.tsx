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
      queryClient.invalidateQueries({ queryKey: ['payment-types'] })
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

// Map known backend error strings/prefixes to translation keys
const errorMap: [RegExp, string][] = [
  [/^connection failed:/, 'error.smtp_connection_failed'],
  [/^failed to decrypt saved password$/, 'error.smtp_decrypt_failed'],
  [/^password is required$/, 'error.password_required'],
  [/^host, username, and from_email are required$/, 'error.smtp_fields_required'],
  [/^account_number or iban is required$/, 'error.account_or_iban_required'],
  [/^invoice number already exists:/, 'error.invoice_number_exists'],
  [/^supplier_id is required$/, 'error.supplier_required'],
  [/^customer_id and supplier_id are required$/, 'error.supplier_customer_required'],
  [/^bank_account_id is required for this payment method$/, 'error.bank_required_for_payment'],
  [/^name is required$/, 'error.name_required'],
  [/^PDF generation failed:/, 'error.pdf_generation_failed'],
  [/^passphrase must be at least 8 characters$/, 'error.passphrase_too_short'],
  [/^file is encrypted, passphrase required$/, 'error.passphrase_required'],
]

export function translateError(t: (key: string) => string, message: string): string {
  for (const [pattern, key] of errorMap) {
    if (pattern.test(message)) {
      const translated = t(key)
      if (translated !== key) return translated
    }
  }
  return message
}
