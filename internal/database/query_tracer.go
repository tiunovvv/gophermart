package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type queryTracer struct {
	log *zap.SugaredLogger
}

func newQueryTracer(logger *zap.SugaredLogger) *queryTracer {
	return &queryTracer{logger}
}

func (t *queryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	t.log.Infof("Running query %s (%v)", data.SQL, data.Args)
	return ctx
}

func (t *queryTracer) TraceQueryEnd(_ context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	t.log.Infof("%v", data.CommandTag)
}
