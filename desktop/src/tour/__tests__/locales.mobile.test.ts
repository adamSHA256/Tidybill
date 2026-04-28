import { describe, it, expect } from 'vitest'
import cs from '../../locales/cs.json'
import sk from '../../locales/sk.json'
import en from '../../locales/en.json'

const MOBILE_KEYS = [
  'tour.create_invoice.mobile.step1.body',
  'tour.just_show_me.mobile.step2.body',
  'tour.just_show_me.mobile.step7.body',
] as const

const locales = { cs, sk, en } as const

describe('locale parity for mobile-tour keys', () => {
  for (const lang of Object.keys(locales) as (keyof typeof locales)[]) {
    for (const key of MOBILE_KEYS) {
      it(`${lang}.json defines "${key}"`, () => {
        const dict = locales[lang] as Record<string, string>
        expect(dict[key], `missing in ${lang}.json`).toBeTruthy()
      })
    }
  }
})
