package mocks

import (
	"github.com/muety/wakapi/models"
	"github.com/stretchr/testify/mock"
)

type BrandingServiceMock struct {
	mock.Mock
}

func (m *BrandingServiceMock) Get() *models.Branding {
	args := m.Called()
	return args.Get(0).(*models.Branding)
}

func (m *BrandingServiceMock) Defaults() models.Branding {
	args := m.Called()
	return args.Get(0).(models.Branding)
}

func (m *BrandingServiceMock) GetOverrides() *models.Branding {
	args := m.Called()
	return args.Get(0).(*models.Branding)
}

func (m *BrandingServiceMock) Set(b *models.Branding) ([]string, error) {
	args := m.Called(b)
	return args.Get(0).([]string), args.Error(1)
}

func (m *BrandingServiceMock) Revision() uint64 {
	args := m.Called()
	return uint64(args.Int(0))
}

// NewStaticBrandingServiceMock returns a mock that always answers with the given
// branding, which is what most handler tests want: branding present, not asserted on.
func NewStaticBrandingServiceMock(branding *models.Branding) *BrandingServiceMock {
	m := new(BrandingServiceMock)
	m.On("Get").Return(branding)
	m.On("Defaults").Return(*branding)
	m.On("GetOverrides").Return(&models.Branding{})
	m.On("Revision").Return(0)
	return m
}
