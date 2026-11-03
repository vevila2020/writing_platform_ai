package app

import (
	"context"
	statisticsapp "github.com/vevila2020/writing_platform_ai/internal/app/statistics"
	"time"
)

func (a *App) Statistics(ctx context.Context, projectID string, days int, location *time.Location) (statisticsapp.Report, error) {
	a.ensureServices()
	return a.statisticsApp.Query(ctx, projectID, days, location, time.Now())
}
