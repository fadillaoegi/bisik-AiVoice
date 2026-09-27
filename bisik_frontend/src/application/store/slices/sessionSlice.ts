import { createSlice, type PayloadAction } from '@reduxjs/toolkit'
import type { Session } from '../../../domain/entities/session'

interface SessionState {
  current: Session | null
  connected: boolean
  recording: boolean
  error: string | null
}

const initialState: SessionState = {
  current: null,
  connected: false,
  recording: false,
  error: null,
}

const sessionSlice = createSlice({
  name: 'session',
  initialState,
  reducers: {
    sessionStarted(state, action: PayloadAction<Session>) {
      state.current = action.payload
      state.error = null
    },
    sessionEnded(state, action: PayloadAction<Session>) {
      state.current = action.payload
      state.recording = false
    },
    connectionChanged(state, action: PayloadAction<boolean>) {
      state.connected = action.payload
    },
    recordingChanged(state, action: PayloadAction<boolean>) {
      state.recording = action.payload
    },
    sessionFailed(state, action: PayloadAction<string>) {
      state.error = action.payload
      state.recording = false
    },
    sessionReset() {
      return initialState
    },
  },
})

export const {
  sessionStarted,
  sessionEnded,
  connectionChanged,
  recordingChanged,
  sessionFailed,
  sessionReset,
} = sessionSlice.actions

export default sessionSlice.reducer
