// Copyright 2019 TiKV Project Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License

package cluster

import (
	. "github.com/pingcap/check"
	"github.com/pingcap/kvproto/pkg/pdpb"
	"github.com/tikv/pd/server/config"
	"github.com/tikv/pd/server/config2"
	"github.com/tikv/pd/server/schedule/storelimit"
)

var _ = Suite(&testStoreLimiterSuite{})

type testStoreLimiterSuite struct {
	opt *config.PersistOptions
}

func (s *testStoreLimiterSuite) SetUpSuite(c *C) {
	// Create a server for testing
	s.opt = config.NewPersistOptions(config.NewConfig())
}

func (s *testStoreLimiterSuite) TestCollect(c *C) {
	limiter := NewStoreLimiter(s.opt)

	limiter.Collect(&pdpb.StoreStats{})
	c.Assert(limiter.state.cst.total, Equals, int64(1))
}

func (s *testStoreLimiterSuite) TestStoreLimitScene(c *C) {
	limiter := NewStoreLimiter(s.opt)
	c.Assert(limiter.scene[config2.AddPeer], DeepEquals, storelimit.DefaultScene(config2.AddPeer))
	c.Assert(limiter.scene[config2.RemovePeer], DeepEquals, storelimit.DefaultScene(config2.RemovePeer))
}

func (s *testStoreLimiterSuite) TestReplaceStoreLimitScene(c *C) {
	limiter := NewStoreLimiter(s.opt)

	sceneAddPeer := &storelimit.Scene{Idle: 4, Low: 3, Normal: 2, High: 1}
	limiter.ReplaceStoreLimitScene(sceneAddPeer, config2.AddPeer)

	c.Assert(limiter.scene[config2.AddPeer], DeepEquals, sceneAddPeer)

	sceneRemovePeer := &storelimit.Scene{Idle: 5, Low: 4, Normal: 3, High: 2}
	limiter.ReplaceStoreLimitScene(sceneRemovePeer, config2.RemovePeer)
}
