package mocks

import (
	"github.com/stretchr/testify/mock"

	"github.com/muety/wakapi/models"
	"github.com/muety/wakapi/utils"
)

type LeaderboardServiceMock struct {
	mock.Mock
}

func (m *LeaderboardServiceMock) GetDefaultScope() *models.IntervalKey {
	args := m.Called()
	if args.Get(0) != nil {
		return args.Get(0).(*models.IntervalKey)
	}
	return nil
}

func (m *LeaderboardServiceMock) Schedule() {
	m.Called()
}

func (m *LeaderboardServiceMock) ComputeLeaderboard(u []*models.User, i *models.IntervalKey, t []uint8) error {
	args := m.Called(u, i, t)
	return args.Error(0)
}

func (m *LeaderboardServiceMock) ExistsAnyByUser(s string) (bool, error) {
	args := m.Called(s)
	return args.Bool(0), args.Error(1)
}

func (m *LeaderboardServiceMock) CountUsers(b bool) (int64, error) {
	args := m.Called(b)
	return int64(args.Int(0)), args.Error(1)
}

func (m *LeaderboardServiceMock) GetByInterval(i *models.IntervalKey, p *utils.PageParams, b bool) (models.Leaderboard, error) {
	args := m.Called(i, p, b)
	if args.Get(0) != nil {
		return args.Get(0).(models.Leaderboard), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *LeaderboardServiceMock) GetByIntervalAndUser(i *models.IntervalKey, s string, b bool) (models.Leaderboard, error) {
	args := m.Called(i, s, b)
	if args.Get(0) != nil {
		return args.Get(0).(models.Leaderboard), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *LeaderboardServiceMock) GetAggregatedByInterval(i *models.IntervalKey, t *uint8, p *utils.PageParams, b bool) (models.Leaderboard, error) {
	args := m.Called(i, t, p, b)
	if args.Get(0) != nil {
		return args.Get(0).(models.Leaderboard), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *LeaderboardServiceMock) GetAggregatedByIntervalAndUser(i *models.IntervalKey, s string, t *uint8, b bool) (models.Leaderboard, error) {
	args := m.Called(i, s, t, b)
	if args.Get(0) != nil {
		return args.Get(0).(models.Leaderboard), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *LeaderboardServiceMock) GenerateByUser(u *models.User, i *models.IntervalKey) (*models.LeaderboardItem, error) {
	args := m.Called(u, i)
	if args.Get(0) != nil {
		return args.Get(0).(*models.LeaderboardItem), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *LeaderboardServiceMock) GenerateAggregatedByUser(u *models.User, i *models.IntervalKey, t uint8) ([]*models.LeaderboardItem, error) {
	args := m.Called(u, i, t)
	if args.Get(0) != nil {
		return args.Get(0).([]*models.LeaderboardItem), args.Error(1)
	}
	return nil, args.Error(1)
}
