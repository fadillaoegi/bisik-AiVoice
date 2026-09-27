import { configureStore } from '@reduxjs/toolkit'
import authReducer from './slices/authSlice'
import sessionReducer from './slices/sessionSlice'
import transcriptReducer from './slices/transcriptSlice'
import complianceReducer from './slices/complianceSlice'

export const store = configureStore({
  reducer: {
    auth: authReducer,
    session: sessionReducer,
    transcript: transcriptReducer,
    compliance: complianceReducer,
  },
  middleware: (getDefault) =>
    // Frame audio tidak pernah masuk store, jadi cek serializable tetap aman.
    getDefault({ serializableCheck: true }),
})

export type RootState = ReturnType<typeof store.getState>
export type AppDispatch = typeof store.dispatch
