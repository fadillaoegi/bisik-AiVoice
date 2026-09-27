import { useDispatch, useSelector } from 'react-redux'
import type { AppDispatch, RootState } from '../../application/store'
import type { TypedUseSelectorHook } from 'react-redux'

export const useAppDispatch: () => AppDispatch = useDispatch
export const useAppSelector: TypedUseSelectorHook<RootState> = useSelector
