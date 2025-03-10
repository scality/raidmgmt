package logicalvolumegetter_test

import "github.com/stretchr/testify/mock"

type MockCommandRunner struct {
	mock.Mock
}

func (m *MockCommandRunner) Run(args []string) ([]byte, error) {
	arguments := m.Called(args)
	return arguments.Get(0).([]byte), arguments.Error(1)
}
