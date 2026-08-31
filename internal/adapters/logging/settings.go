package logging

import "github.com/adsouza/africa2ice/pkg/ui"

type settingsDecorator struct {
	session *Session
	next    ui.UISettingsStore
}

func DecorateUISettingsStore(session *Session, next ui.UISettingsStore) ui.UISettingsStore {
	if session == nil || next == nil {
		return next
	}
	return &settingsDecorator{session: session, next: next}
}

func (decorator *settingsDecorator) BeginRead(revision uint64) error {
	id, started := decorator.session.start("settings", "read", "settings_revision", revision)
	err := decorator.next.BeginRead(revision)
	decorator.session.end(id, started, "settings", "read", err, "settings_revision", revision)
	return err
}

func (decorator *settingsDecorator) BeginWrite(revision uint64, settings ui.UISettings) error {
	id, started := decorator.session.start("settings", "write", "settings_revision", revision, "schema_version", settings.SchemaVersion)
	err := decorator.next.BeginWrite(revision, settings)
	decorator.session.end(id, started, "settings", "write", err, "settings_revision", revision, "schema_version", settings.SchemaVersion)
	return err
}

func (decorator *settingsDecorator) Poll() []ui.UISettingsCompletion {
	completions := decorator.next.Poll()
	for _, completion := range completions {
		attributes := []any{
			"operation", completion.Operation,
			"settings_revision", completion.Revision,
			"schema_version", completion.Settings.SchemaVersion,
			"outcome", "ok",
		}
		if completion.Err != nil {
			attributes[7] = "error"
			attributes = append(attributes, "error", completion.Err.Error())
		}
		decorator.session.logger.Info("settings.completion", attributes...)
	}
	return completions
}

var _ ui.UISettingsStore = (*settingsDecorator)(nil)
