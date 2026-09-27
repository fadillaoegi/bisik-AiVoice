import { createSlice, type PayloadAction } from '@reduxjs/toolkit'
import type { Speaker, Utterance } from '../../../domain/entities/session'

interface TranscriptState {
  utterances: Utterance[]
  partial: { speaker: Speaker; text: string } | null
}

const initialState: TranscriptState = { utterances: [], partial: null }

const transcriptSlice = createSlice({
  name: 'transcript',
  initialState,
  reducers: {
    partialReceived(state, action: PayloadAction<{ speaker: Speaker; text: string }>) {
      state.partial = action.payload
    },
    utteranceAppended(state, action: PayloadAction<Utterance>) {
      state.partial = null
      const idx = state.utterances.findIndex((u) => u.id === action.payload.id)
      if (idx >= 0) state.utterances[idx] = action.payload
      else state.utterances.push(action.payload)
    },
    /**
     * Diarization merevisi label pembicara sebelumnya.
     * Ini fitur, bukan bug — tandai `revised` supaya bisa ditonjolkan di UI.
     */
    speakerRevised(state, action: PayloadAction<{ id: string; speaker: Speaker }>) {
      const u = state.utterances.find((x) => x.id === action.payload.id)
      if (u) {
        u.speaker = action.payload.speaker
        u.revised = true
      }
    },
    /** Mengisi transkrip dari snapshot sesi yang sudah berjalan. */
    transcriptLoaded(state, action: PayloadAction<Utterance[]>) {
      state.utterances = action.payload
      state.partial = null
    },
    transcriptCleared() {
      return initialState
    },
  },
})

export const {
  partialReceived,
  utteranceAppended,
  speakerRevised,
  transcriptLoaded,
  transcriptCleared,
} = transcriptSlice.actions

export default transcriptSlice.reducer
