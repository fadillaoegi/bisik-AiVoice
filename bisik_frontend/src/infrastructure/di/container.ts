import { HttpAuthRepository } from '../api/http_auth_repository'
import { HttpSessionRepository } from '../api/http_session_repository'
import { WorkletAudioRepository } from '../audio/worklet_audio_repository'
import { WebSpeechRepository } from '../audio/web_speech_repository'
import { SessionSocket } from '../ws/session_socket'
import { StartSessionUseCase } from '../../application/usecases/start_session'
import { EndSessionUseCase } from '../../application/usecases/end_session'
import { GetReportUseCase } from '../../application/usecases/get_report'
import { StreamAudioUseCase } from '../../application/usecases/stream_audio'

/**
 * Composition root: satu-satunya tempat implementasi konkret dirakit.
 * Layer presentation hanya menyentuh use case, tidak pernah repository.
 */
const sessionRepository = new HttpSessionRepository()
const audioRepository = new WorkletAudioRepository()

export const container = {
  repositories: {
    session: sessionRepository,
    audio: audioRepository,
    speech: new WebSpeechRepository(),
    auth: new HttpAuthRepository(),
  },
  usecases: {
    startSession: new StartSessionUseCase(sessionRepository),
    endSession: new EndSessionUseCase(sessionRepository),
    getReport: new GetReportUseCase(sessionRepository),
    streamAudio: new StreamAudioUseCase(audioRepository),
  },
  socket: new SessionSocket(),
}

export type Container = typeof container
