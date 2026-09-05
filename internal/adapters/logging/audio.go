package logging

// LogAudioFailure records one audio-subsystem failure. The device error is
// otherwise invisible in a session log: a failed sound device used to abort
// the Ebitengine loop and reach stderr only, so a submitted log showed an
// orderly session.end and no cause. stage is "init" for a device that never
// opened and "play" for one that stopped working.
func (session *Session) LogAudioFailure(stage string, err error) {
	if session == nil {
		return
	}
	message := "unknown"
	if err != nil {
		message = err.Error()
	}
	session.logger.Error("audio.failure", "stage", stage, "error", message)
}
