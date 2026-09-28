import { createContext } from 'react'
import { DEFAULT_LANG, dictionaries, type Dictionary, type Lang } from './dictionary'

export interface I18nValue {
  lang: Lang
  t: Dictionary
  setLang: (lang: Lang) => void
}

export const I18nContext = createContext<I18nValue>({
  lang: DEFAULT_LANG,
  t: dictionaries[DEFAULT_LANG],
  setLang: () => {},
})
