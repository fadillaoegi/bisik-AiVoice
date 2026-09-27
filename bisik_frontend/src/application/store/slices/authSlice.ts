import { createSlice, type PayloadAction } from '@reduxjs/toolkit'
import type { AuthUser } from '../../../domain/entities/auth'

interface AuthState {
  user: AuthUser | null
  /** Pemulihan sesi login dari token tersimpan belum selesai. */
  restoring: boolean
  error: string | null
}

const initialState: AuthState = { user: null, restoring: true, error: null }

const authSlice = createSlice({
  name: 'auth',
  initialState,
  reducers: {
    authRestored(state, action: PayloadAction<AuthUser | null>) {
      state.user = action.payload
      state.restoring = false
    },
    loggedIn(state, action: PayloadAction<AuthUser>) {
      state.user = action.payload
      state.error = null
    },
    loginFailed(state, action: PayloadAction<string>) {
      state.user = null
      state.error = action.payload
    },
    loggedOut() {
      return { ...initialState, restoring: false }
    },
  },
})

export const { authRestored, loggedIn, loginFailed, loggedOut } = authSlice.actions
export default authSlice.reducer
