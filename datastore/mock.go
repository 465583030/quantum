// Package datastore mock struct and func's
// Copyright (c) 2016 Christian Saide <Supernomad>
// Licensed under the MPL-2.0, for details see https://github.com/Supernomad/quantum/blob/master/LICENSE
package datastore

import (
	"sync"

	"github.com/Supernomad/quantum/common"
)

// Mock datastore
type Mock struct {
	InternalMapping *common.Mapping

	wg *sync.WaitGroup
}

// Mapping from the mock datastore
func (mock *Mock) Mapping(ip uint32) (*common.Mapping, bool) {
	return mock.InternalMapping, true
}

// Init the mock datastore
func (mock *Mock) Init() error {
	return nil
}

// Start the mock datastore
func (mock *Mock) Start(wg *sync.WaitGroup) {
	mock.wg = wg
}

// Stop the mock datastore
func (mock *Mock) Stop() {
	mock.wg.Done()
}

func newMock(log *common.Logger, cfg *common.Config) (Datastore, error) {
	return &Mock{}, nil
}
