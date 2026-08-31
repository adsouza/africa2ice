package logging

import "github.com/adsouza/africa2ice/internal/application"

type repositoryDecorator struct {
	session *Session
	next    application.CampaignRepository
}

func DecorateCampaignRepository(session *Session, next application.CampaignRepository) application.CampaignRepository {
	if session == nil || next == nil {
		return next
	}
	return &repositoryDecorator{session: session, next: next}
}

func (decorator *repositoryDecorator) BeginWrite(operation application.RepositoryOpID, slot application.SlotID, state application.SaveState) error {
	return decorator.begin("write", operation, slot, func() error { return decorator.next.BeginWrite(operation, slot, state) })
}

func (decorator *repositoryDecorator) BeginRead(operation application.RepositoryOpID, slot application.SlotID) error {
	return decorator.begin("read", operation, slot, func() error { return decorator.next.BeginRead(operation, slot) })
}

func (decorator *repositoryDecorator) BeginDelete(operation application.RepositoryOpID, slot application.SlotID) error {
	return decorator.begin("delete", operation, slot, func() error { return decorator.next.BeginDelete(operation, slot) })
}

func (decorator *repositoryDecorator) BeginList(operation application.RepositoryOpID) error {
	return decorator.begin("list", operation, 0, func() error { return decorator.next.BeginList(operation) })
}

func (decorator *repositoryDecorator) begin(name string, operation application.RepositoryOpID, slot application.SlotID, call func() error) error {
	id, started := decorator.session.start("repository", name, "repository_operation_id", uint64(operation), "slot", int(slot))
	err := call()
	decorator.session.end(id, started, "repository", name, err, "repository_operation_id", uint64(operation), "slot", int(slot))
	return err
}

func (decorator *repositoryDecorator) Poll() []application.RepositoryCompletion {
	completions := decorator.next.Poll()
	for _, completion := range completions {
		outcome := "ok"
		attributes := []any{
			"repository_operation_id", uint64(completion.OperationID),
			"operation", completion.Operation,
			"slot", int(completion.SlotID),
			"outcome", outcome,
		}
		if completion.Err != nil {
			attributes[7] = "error"
			attributes = append(attributes, "error", completion.Err.Error())
		}
		decorator.session.logger.Info("repository.completion", attributes...)
	}
	return completions
}

func (decorator *repositoryDecorator) Writable() bool { return decorator.next.Writable() }
func (decorator *repositoryDecorator) Close() error   { return decorator.next.Close() }

var _ application.CampaignRepository = (*repositoryDecorator)(nil)
