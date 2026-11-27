package natsadapter

import (
	"context"

	"github.com/drukleague/platform/packages/eventkit"
	"github.com/drukleague/platform/services/search/internal/domain"
)

type IndexWriter interface {
	ApplyIndexEvent(context.Context, domain.IndexEvent) error
}

func Handler(writer IndexWriter, entityType string) eventkit.Handler {
	return func(ctx context.Context, _ string, payload []byte) error {
		event, err := domain.DecodeIndexEvent(payload, entityType)
		if err != nil {
			return eventkit.Permanent(err)
		}
		return writer.ApplyIndexEvent(ctx, event)
	}
}

func Consumers(writer IndexWriter) []eventkit.ConsumerConfig {
	return []eventkit.ConsumerConfig{
		{Durable: "search-team-index-v1", Subject: eventkit.SubjectSearchTeam, Handler: Handler(writer, domain.EntityTeam)},
		{Durable: "search-player-index-v1", Subject: eventkit.SubjectSearchPlayer, Handler: Handler(writer, domain.EntityPlayer)},
		{Durable: "search-tournament-index-v1", Subject: eventkit.SubjectSearchTournament, Handler: Handler(writer, domain.EntityTournament)},
	}
}
