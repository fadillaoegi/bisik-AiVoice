import { createSlice, type PayloadAction } from '@reduxjs/toolkit'
import type { Obligation, Violation } from '../../../domain/entities/compliance'

interface ComplianceState {
  obligations: Obligation[]
  violations: Violation[]
  lastNudge: string | null
}

const initialState: ComplianceState = { obligations: [], violations: [], lastNudge: null }

const complianceSlice = createSlice({
  name: 'compliance',
  initialState,
  reducers: {
    obligationsLoaded(state, action: PayloadAction<Obligation[]>) {
      state.obligations = action.payload
    },
    obligationSatisfied(
      state,
      action: PayloadAction<{ code: string; confidence: number; evidenceId: string }>,
    ) {
      const ob = state.obligations.find((o) => o.code === action.payload.code)
      if (ob) {
        ob.status = 'satisfied'
        ob.confidence = action.payload.confidence
        ob.evidenceId = action.payload.evidenceId
      }
    },
    violationDetected(state, action: PayloadAction<Violation>) {
      state.violations.push(action.payload)
    },
    /**
     * Mengisi pelanggaran dari snapshot, bukan menambah satu per satu.
     * Dipakai supervisor yang bergabung di tengah sesi: tanpa ini dia hanya
     * melihat pelanggaran yang terjadi SETELAH dia menyambung.
     */
    violationsLoaded(state, action: PayloadAction<Violation[]>) {
      state.violations = action.payload
    },
    nudgeReceived(state, action: PayloadAction<string>) {
      state.lastNudge = action.payload
    },
    complianceReset() {
      return initialState
    },
  },
})

export const {
  obligationsLoaded,
  obligationSatisfied,
  violationDetected,
  violationsLoaded,
  nudgeReceived,
  complianceReset,
} = complianceSlice.actions

export default complianceSlice.reducer
