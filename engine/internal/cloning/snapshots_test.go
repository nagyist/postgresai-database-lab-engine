/*
2021 © Postgres.ai
*/

package cloning

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

func (s *BaseCloningSuite) TestLatestSnapshot() {
	s.cloning.resetSnapshots(make(map[string]*models.Snapshot), nil)

	snapshot1 := &models.Snapshot{
		ID:          "TestSnapshotID1",
		CreatedAt:   &models.LocalTime{Time: time.Date(2020, 02, 20, 01, 23, 45, 0, time.UTC)},
		DataStateAt: &models.LocalTime{Time: time.Date(2020, 02, 19, 0, 0, 0, 0, time.UTC)},
	}

	snapshot2 := &models.Snapshot{
		ID:          "TestSnapshotID2",
		CreatedAt:   &models.LocalTime{Time: time.Date(2020, 02, 20, 05, 43, 21, 0, time.UTC)},
		DataStateAt: &models.LocalTime{Time: time.Date(2020, 02, 20, 0, 0, 0, 0, time.UTC)},
	}

	require.Equal(s.T(), 0, len(s.cloning.snapshotBox.items))
	latestSnapshot, err := s.cloning.getLatestSnapshot()
	require.Nil(s.T(), latestSnapshot)
	require.EqualError(s.T(), err, "no snapshot found")

	s.cloning.addSnapshot(snapshot1)
	s.cloning.addSnapshot(snapshot2)

	latestSnapshot, err = s.cloning.getLatestSnapshot()
	require.NoError(s.T(), err)
	require.Equal(s.T(), latestSnapshot, snapshot2)

	snapshot3 := &models.Snapshot{
		ID:          "TestSnapshotID3",
		CreatedAt:   &models.LocalTime{Time: time.Date(2020, 02, 21, 05, 43, 21, 0, time.UTC)},
		DataStateAt: &models.LocalTime{Time: time.Date(2020, 02, 21, 0, 0, 0, 0, time.UTC)},
	}

	snapshotMap := make(map[string]*models.Snapshot)
	snapshotMap[snapshot1.ID] = snapshot1
	snapshotMap[snapshot2.ID] = snapshot2
	snapshotMap[snapshot3.ID] = snapshot3
	s.cloning.resetSnapshots(snapshotMap, snapshot3)

	require.Equal(s.T(), 3, len(s.cloning.snapshotBox.items))
	latestSnapshot, err = s.cloning.getLatestSnapshot()
	require.NoError(s.T(), err)
	require.Equal(s.T(), latestSnapshot, snapshot3)
}

func (s *BaseCloningSuite) TestSnapshotByID() {
	s.cloning.resetSnapshots(make(map[string]*models.Snapshot), nil)

	snapshot1 := &models.Snapshot{
		ID:          "TestSnapshotID1",
		CreatedAt:   &models.LocalTime{Time: time.Date(2020, 02, 20, 01, 23, 45, 0, time.UTC)},
		DataStateAt: &models.LocalTime{Time: time.Date(2020, 02, 19, 0, 0, 0, 0, time.UTC)},
	}

	snapshot2 := &models.Snapshot{
		ID:          "TestSnapshotID2",
		CreatedAt:   &models.LocalTime{Time: time.Date(2020, 02, 20, 05, 43, 21, 0, time.UTC)},
		DataStateAt: &models.LocalTime{Time: time.Date(2020, 02, 20, 0, 0, 0, 0, time.UTC)},
	}

	require.Equal(s.T(), 0, len(s.cloning.snapshotBox.items))
	latestSnapshot, err := s.cloning.getLatestSnapshot()
	require.Nil(s.T(), latestSnapshot)
	require.EqualError(s.T(), err, "no snapshot found")

	s.cloning.addSnapshot(snapshot1)
	require.Equal(s.T(), 1, len(s.cloning.snapshotBox.items))
	require.Equal(s.T(), "TestSnapshotID1", s.cloning.snapshotBox.items[snapshot1.ID].ID)

	s.cloning.addSnapshot(snapshot2)
	require.Equal(s.T(), 2, len(s.cloning.snapshotBox.items))
	require.Equal(s.T(), "TestSnapshotID2", s.cloning.snapshotBox.items[snapshot2.ID].ID)

	latestSnapshot, err = s.cloning.getSnapshotByID("TestSnapshotID2")
	require.NoError(s.T(), err)
	require.Equal(s.T(), latestSnapshot, snapshot2)

	s.cloning.resetSnapshots(make(map[string]*models.Snapshot), nil)
	require.Equal(s.T(), 0, len(s.cloning.snapshotBox.items))
	latestSnapshot, err = s.cloning.getLatestSnapshot()
	require.Nil(s.T(), latestSnapshot)
	require.EqualError(s.T(), err, "no snapshot found")
}

func TestCloneCounter(t *testing.T) {
	c := &Base{}
	c.snapshotBox.items = make(map[string]*models.Snapshot)
	snapshot := &models.Snapshot{
		ID:        "testSnapshotID",
		NumClones: 0,
	}
	c.snapshotBox.items[snapshot.ID] = snapshot

	snapshot, err := c.getSnapshotByID("testSnapshotID")
	require.Nil(t, err)
	require.Equal(t, 0, snapshot.NumClones)

	c.IncrementCloneNumber("testSnapshotID")
	snapshot, err = c.getSnapshotByID("testSnapshotID")
	require.Nil(t, err)
	require.Equal(t, 1, snapshot.NumClones)

	c.decrementCloneNumber("testSnapshotID")
	snapshot, err = c.getSnapshotByID("testSnapshotID")
	require.Nil(t, err)
	require.Equal(t, 0, snapshot.NumClones)
}

func TestInitialCloneCounter(t *testing.T) {
	c := &Base{clones: make(map[string]*CloneWrapper)}

	snapshot := &models.Snapshot{
		ID: "testSnapshotID",
	}

	snapshot2 := &models.Snapshot{
		ID: "testSnapshotID2",
	}

	cloneWrapper01 := &CloneWrapper{
		Clone: &models.Clone{
			ID:       "test_clone001",
			Snapshot: snapshot,
		},
	}

	cloneWrapper02 := &CloneWrapper{
		Clone: &models.Clone{
			ID:       "test_clone002",
			Snapshot: snapshot,
		},
	}

	cloneWrapper03 := &CloneWrapper{
		Clone: &models.Clone{
			ID:       "test_clone003",
			Snapshot: snapshot2,
		},
	}

	c.setWrapper("test_clone001", cloneWrapper01)
	c.setWrapper("test_clone002", cloneWrapper02)
	c.setWrapper("test_clone003", cloneWrapper03)

	counters := c.counterClones()

	require.Len(t, counters, 2)
	require.Len(t, counters["testSnapshotID"], 2)
	require.Len(t, counters["testSnapshotID2"], 1)
	require.Len(t, counters["testSnapshotID3"], 0)
	require.ElementsMatch(t, []string{"test_clone001", "test_clone002"}, counters["testSnapshotID"])
}

func (s *BaseCloningSuite) TestGetSnapshotList() {
	s.cloning.resetSnapshots(make(map[string]*models.Snapshot), nil)

	snap1 := &models.Snapshot{
		ID:        "snap1",
		CreatedAt: &models.LocalTime{Time: time.Date(2020, 1, 10, 0, 0, 0, 0, time.UTC)},
	}
	snap2 := &models.Snapshot{
		ID:        "snap2",
		CreatedAt: &models.LocalTime{Time: time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC)},
	}
	snap3 := &models.Snapshot{
		ID:        "snap3",
		CreatedAt: &models.LocalTime{Time: time.Date(2020, 2, 20, 0, 0, 0, 0, time.UTC)},
	}

	s.cloning.addSnapshot(snap1)
	s.cloning.addSnapshot(snap2)
	s.cloning.addSnapshot(snap3)

	list := s.cloning.getSnapshotList()
	require.Len(s.T(), list, 3)
	require.Equal(s.T(), "snap2", list[0].ID, "newest snapshot should be first")
	require.Equal(s.T(), "snap3", list[1].ID)
	require.Equal(s.T(), "snap1", list[2].ID, "oldest snapshot should be last")
}

func (s *BaseCloningSuite) TestGetSnapshotListEmpty() {
	s.cloning.resetSnapshots(make(map[string]*models.Snapshot), nil)

	list := s.cloning.getSnapshotList()
	require.Empty(s.T(), list)
}

func (s *BaseCloningSuite) TestHasDependentSnapshots() {
	s.cloning.resetSnapshots(make(map[string]*models.Snapshot), nil)

	wrapper := &CloneWrapper{
		Clone: &models.Clone{
			ID:       "myclone",
			Branch:   "main",
			Revision: 0,
			Snapshot: &models.Snapshot{Pool: "pool1"},
		},
	}

	require.False(s.T(), s.cloning.hasDependentSnapshots(wrapper), "no snapshots means no dependents")

	unrelatedSnap := &models.Snapshot{ID: "other/branch/data@snap1", CreatedAt: &models.LocalTime{Time: time.Now()}, DataStateAt: &models.LocalTime{Time: time.Now()}}
	s.cloning.addSnapshot(unrelatedSnap)
	require.False(s.T(), s.cloning.hasDependentSnapshots(wrapper), "unrelated snapshot should not match")

	dependentSnap := &models.Snapshot{
		ID:          "pool1/branch/main/myclone/r0@snap_dep",
		CreatedAt:   &models.LocalTime{Time: time.Now()},
		DataStateAt: &models.LocalTime{Time: time.Now()},
	}
	s.cloning.addSnapshot(dependentSnap)
	require.True(s.T(), s.cloning.hasDependentSnapshots(wrapper), "snapshot with matching prefix should be dependent")
}

func (s *BaseCloningSuite) TestHasDependentSnapshotsRevision() {
	s.cloning.resetSnapshots(make(map[string]*models.Snapshot), nil)

	wrapper := &CloneWrapper{
		Clone: &models.Clone{
			ID:       "myclone",
			Branch:   "main",
			Revision: 1,
			Snapshot: &models.Snapshot{Pool: "pool1"},
		},
	}

	r0Snap := &models.Snapshot{
		ID:          "pool1/branch/main/myclone/r0@snap1",
		CreatedAt:   &models.LocalTime{Time: time.Now()},
		DataStateAt: &models.LocalTime{Time: time.Now()},
	}
	s.cloning.addSnapshot(r0Snap)
	require.False(s.T(), s.cloning.hasDependentSnapshots(wrapper), "r0 snapshot should not match r1 clone")

	r1Snap := &models.Snapshot{
		ID:          "pool1/branch/main/myclone/r1@snap2",
		CreatedAt:   &models.LocalTime{Time: time.Now()},
		DataStateAt: &models.LocalTime{Time: time.Now()},
	}
	s.cloning.addSnapshot(r1Snap)
	require.True(s.T(), s.cloning.hasDependentSnapshots(wrapper), "r1 snapshot should match r1 clone")
}

func TestGetCloneNumber(t *testing.T) {
	c := &Base{}
	c.snapshotBox.items = make(map[string]*models.Snapshot)
	c.snapshotBox.items["snap1"] = &models.Snapshot{ID: "snap1", NumClones: 5}

	require.Equal(t, 5, c.GetCloneNumber("snap1"))
	require.Equal(t, 0, c.GetCloneNumber("nonexistent"))
}

func TestDecrementCloneNumberAtZero(t *testing.T) {
	c := &Base{}
	c.snapshotBox.items = make(map[string]*models.Snapshot)
	c.snapshotBox.items["snap1"] = &models.Snapshot{ID: "snap1", NumClones: 0}

	c.decrementCloneNumber("snap1")
	require.Equal(t, 0, c.snapshotBox.items["snap1"].NumClones, "should not go negative")
}

func TestIncrementDecrementCloneNumberMissingSnapshot(t *testing.T) {
	c := &Base{}
	c.snapshotBox.items = make(map[string]*models.Snapshot)

	c.IncrementCloneNumber("nonexistent")
	c.decrementCloneNumber("nonexistent")
	require.Empty(t, c.snapshotBox.items)
}

func TestCounterClonesWithNilValues(t *testing.T) {
	c := &Base{clones: make(map[string]*CloneWrapper)}
	c.setWrapper("nil_wrapper", nil)
	c.setWrapper("nil_snapshot", &CloneWrapper{Clone: &models.Clone{ID: "c1"}})
	c.setWrapper("valid", &CloneWrapper{Clone: &models.Clone{ID: "c2", Snapshot: &models.Snapshot{ID: "snap1"}}})

	counters := c.counterClones()
	require.Len(t, counters, 1)
	require.Equal(t, []string{"valid"}, counters["snap1"])
}

func TestDefineLatestSnapshotWithZeroTime(t *testing.T) {
	zeroTime := &models.Snapshot{DataStateAt: &models.LocalTime{Time: time.Time{}}}
	validTime := &models.Snapshot{DataStateAt: &models.LocalTime{Time: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}}

	result := defineLatestSnapshot(zeroTime, validTime)
	require.Equal(t, validTime, result, "zero-time latest should be replaced by valid challenger")
}

func (s *BaseCloningSuite) TestMakeSnapshotRevisionNotFound() {
	s.cloning.resetSnapshots(make(map[string]*models.Snapshot), nil)

	_, err := s.cloning.getSnapshotByID("nonexistent-revision")
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "no snapshot found")
}

func TestLatestSnapshots(t *testing.T) {
	baseSnapshot := &models.Snapshot{
		DataStateAt: &models.LocalTime{Time: time.Date(2020, 02, 19, 0, 0, 0, 0, time.UTC)},
	}
	newSnapshot := &models.Snapshot{
		DataStateAt: &models.LocalTime{Time: time.Date(2020, 02, 21, 0, 0, 0, 0, time.UTC)},
	}
	oldSnapshot := &models.Snapshot{
		DataStateAt: &models.LocalTime{Time: time.Date(2020, 02, 01, 0, 0, 0, 0, time.UTC)},
	}

	testCases := []struct {
		latest, challenger, result *models.Snapshot
	}{
		{
			latest:     baseSnapshot,
			challenger: newSnapshot,
			result:     newSnapshot,
		},
		{
			latest:     baseSnapshot,
			challenger: oldSnapshot,
			result:     baseSnapshot,
		},
		{
			latest:     nil,
			challenger: oldSnapshot,
			result:     oldSnapshot,
		},
		{
			latest:     &models.Snapshot{},
			challenger: oldSnapshot,
			result:     oldSnapshot,
		},
	}

	for _, tc := range testCases {
		require.Equal(t, tc.result, defineLatestSnapshot(tc.latest, tc.challenger))
	}
}
