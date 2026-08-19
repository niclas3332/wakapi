package migrations

import (
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/muety/wakapi/config"
	"github.com/muety/wakapi/models"
	"github.com/muety/wakapi/repositories"
	"github.com/muety/wakapi/services"
)

func init() {
	const name = "20260819-backfill_ai_stats"

	registerPostMigration(migrationFunc{
		name: name,
		// Runs in the background: it reads a user's entire heartbeat history, which on a
		// long-lived instance takes minutes, and nothing else depends on its result.
		// Everything it writes goes into the two new tables, so a failure loses only
		// pre-aggregation - the next nightly run picks it up again.
		background: true,
		f: func(db *gorm.DB, cfg *config.Config) error {
			if hasRun(name, db) {
				return nil
			}

			// Aggregation happens in Go rather than as a GROUP BY. Day boundaries depend
			// on each user's own time zone, which the database does not know, and the
			// heartbeat time column is epoch milliseconds on sqlite but a timestamp
			// elsewhere - so DATE(time) would quietly produce nonsense on sqlite.
			userRepo := repositories.NewUserRepository(db)
			heartbeatRepo := repositories.NewHeartbeatRepository(db)
			languageMappingRepo := repositories.NewLanguageMappingRepository(db)
			aiStatsRepo := repositories.NewAIStatsRepository(db)
			keyValueRepo := repositories.NewKeyValueRepository(db)

			keyValueService := services.NewKeyValueService(keyValueRepo)
			languageMappingService := services.NewLanguageMappingService(languageMappingRepo)
			heartbeatService := services.NewHeartbeatService(heartbeatRepo, languageMappingService)
			aiStatsService := services.NewAIStatsService(aiStatsRepo, heartbeatService, keyValueService)

			users, err := userRepo.GetAll()
			if err != nil {
				return err
			}

			started := time.Now()
			slog.Info("backfilling ai statistics, this runs in the background and may take a while", "num_users", len(users))

			for _, user := range users {
				if err := backfillUser(aiStatsService, heartbeatService, user); err != nil {
					// One user's history must not stop the rest. The nightly job will
					// retry whatever is still missing.
					config.Log().Error("failed to backfill ai statistics for user", "userID", user.ID, "error", err)
				}
			}

			slog.Info("finished backfilling ai statistics", "time_elapsed", time.Since(started))
			setHasRun(name, db)
			return nil
		},
	})
}

func backfillUser(aiStatsService services.IAIStatsService, heartbeatService services.IHeartbeatService, user *models.User) error {
	first, err := heartbeatService.GetFirstByUser(user)
	if err != nil {
		return err
	}
	if first.IsZero() {
		return nil
	}

	slog.Info("backfilling ai statistics for user", "userID", user.ID, "from", first)
	return aiStatsService.AggregateRange(user, first, time.Now())
}
