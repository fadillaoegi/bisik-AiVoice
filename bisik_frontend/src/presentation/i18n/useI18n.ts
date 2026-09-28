import { useContext } from 'react'
import { I18nContext } from './context'

/** Bahasa aktif, kamusnya, dan cara menggantinya. */
export const useI18n = () => useContext(I18nContext)
