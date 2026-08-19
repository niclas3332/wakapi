package repositories

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/muety/wakapi/models"
)

type AIStatsRepository struct {
	BaseRepository
	db *gorm.DB
}

func NewAIStatsRepository(db *gorm.DB) *AIStatsRepository {
	return &AIStatsRepository{BaseRepository: NewBaseRepository(db), db: db}
}

// GetStatsWithin returns the pre-aggregated rows covering [from, to).
func (r *AIStatsRepository) GetStatsWithin(user *models.User, from, to time.Time) ([]*models.AIStatsDaily, error) {
	var stats []*models.AIStatsDaily
	err := r.db.
		Where("user_id = ?", user.ID).
		Where("date >= ?", models.CustomTime(from)).
		Where("date < ?", models.CustomTime(to)).
		Order("date asc").
		Find(&stats).Error
	return stats, err
}

// GetSessionsWithin returns the session markers covering [from, to). They are separate
// from the stats rows so a session spanning two projects is still counted once.
func (r *AIStatsRepository) GetSessionsWithin(user *models.User, from, to time.Time) ([]*models.AISessionDaily, error) {
	var sessions []*models.AISessionDaily
	err := r.db.
		Where("user_id = ?", user.ID).
		Where("date >= ?", models.CustomTime(from)).
		Where("date < ?", models.CustomTime(to)).
		Find(&sessions).Error
	return sessions, err
}

// ReplaceDay swaps in a freshly computed day. Delete plus insert inside one transaction
// rather than an upsert: a re-run must not leave behind rows for a model or project
// that no longer appears in that day's heartbeats.
func (r *AIStatsRepository) ReplaceDay(userId string, date time.Time, stats []*models.AIStatsDaily, sessions []*models.AISessionDaily) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Where("user_id = ? and date = ?", userId, models.CustomTime(date)).
			Delete(&models.AIStatsDaily{}).Error; err != nil {
			return err
		}
		if err := tx.
			Where("user_id = ? and date = ?", userId, models.CustomTime(date)).
			Delete(&models.AISessionDaily{}).Error; err != nil {
			return err
		}

		if len(stats) > 0 {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&stats).Error; err != nil {
				return err
			}
		}
		if len(sessions) > 0 {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&sessions).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetLastDate reports the newest day already aggregated, so a run can pick up where the
// last one stopped instead of redoing the whole history.
func (r *AIStatsRepository) GetLastDate(userId string) (time.Time, error) {
	var stats models.AIStatsDaily
	err := r.db.
		Where("user_id = ?", userId).
		Order("date desc").
		Limit(1).
		Take(&stats).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}
	return stats.Date.T(), nil
}

func (r *AIStatsRepository) DeleteByUser(userId string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userId).Delete(&models.AIStatsDaily{}).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ?", userId).Delete(&models.AISessionDaily{}).Error
	})
}
