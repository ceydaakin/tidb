// Copyright 2026 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package core

import (
	"testing"
	"time"

	"github.com/pingcap/kvproto/pkg/coprocessor"
	"github.com/pingcap/tidb/pkg/kv"
	"github.com/pingcap/tidb/pkg/parser/mysql"
	"github.com/pingcap/tidb/pkg/tablecodec"
	"github.com/pingcap/tidb/pkg/types"
	"github.com/pingcap/tidb/pkg/util/codec"
	"github.com/stretchr/testify/require"
)

// TestSlowQueryTimeRangeFromKeyspaceKeyRange covers issue 72592: with API v2
// the cop key ranges sent to a TiDB server carry the keyspace prefix, and the
// time range of CLUSTER_SLOW_QUERY must still be decoded from them.
func TestSlowQueryTimeRangeFromKeyspaceKeyRange(t *testing.T) {
	start := time.Date(2200, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2200, 1, 2, 0, 0, 0, 0, time.UTC)
	encodeTimeKey := func(prefix []byte, tm time.Time) []byte {
		d := types.NewTimeDatum(types.NewTime(types.FromGoTime(tm), mysql.TypeDatetime, types.MaxFsp))
		encoded, err := codec.EncodeKey(time.UTC, nil, d)
		require.NoError(t, err)
		handle, err := kv.NewCommonHandle(encoded)
		require.NoError(t, err)
		return append(append([]byte{}, prefix...), tablecodec.EncodeRowKeyWithHandle(1, handle)...)
	}

	for _, prefix := range [][]byte{nil, {'x', 0, 0, 1}} {
		e := &SlowQueryExtractor{}
		err := e.buildTimeRangeFromKeyRange([]*coprocessor.KeyRange{{
			Start: encodeTimeKey(prefix, start),
			End:   encodeTimeKey(prefix, end),
		}})
		require.NoError(t, err)
		require.True(t, e.Enable, "prefix %q", prefix)
		require.Len(t, e.TimeRanges, 1)
		require.True(t, start.Equal(e.TimeRanges[0].StartTime), "prefix %q: got %v", prefix, e.TimeRanges[0].StartTime)
		require.True(t, end.Equal(e.TimeRanges[0].EndTime), "prefix %q: got %v", prefix, e.TimeRanges[0].EndTime)
	}
}
