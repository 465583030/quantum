package inet

import (
	"github.com/Supernomad/quantum/common"
)

// Mock interface
type Mock struct {
}

// Name of the interface
func (mock *Mock) Name() string {
	return "Mocked Interface"
}

// Read a packet off the interface
func (mock *Mock) Read(buf []byte, queue int) (*common.Payload, bool) {
	return common.NewTunPayload(buf, common.MTU), true
}

// Write a packet to the interface
func (mock *Mock) Write(payload *common.Payload, queue int) bool {
	return true
}

// Open the interface
func (mock *Mock) Open() error {
	return nil
}

// Close the interface
func (mock *Mock) Close() error {
	return nil
}

// GetFDs will return the underlying queue fds
func (mock *Mock) GetFDs() []int {
	return nil
}

func newMock(cfg *common.Config) *Mock {
	return &Mock{}
}
