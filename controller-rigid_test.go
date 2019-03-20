package ewp

import (
	"reflect"
	"testing"
)

func TestNewRigidController(t *testing.T) {
	type args struct {
		loadLevels         LoadLevels
		maxChangesPerCycle int
	}
	tests := []struct {
		name    string
		args    args
		want    PoolController
		wantErr bool
	}{
		{name: "1. invalidConfig", args: args{loadLevels: nil, maxChangesPerCycle: -1}, want: nil, wantErr: true},
		{name: "2. nilLoadLevels", args: args{loadLevels: nil, maxChangesPerCycle: 0}, want: &rigidController{levels: defaultLoadLevels, maxChangesPerCycle: 0}},

		{name: "3. defaultLoadLevels", args: args{loadLevels: defaultLoadLevels, maxChangesPerCycle: 2}, want: &rigidController{levels: defaultLoadLevels, maxChangesPerCycle: 2}},
		{name: "4. customLoadLevel_sorted", args: args{loadLevels: []LoadLevel{{0, 0}, {0.5, 0.5}, {1, 1}}, maxChangesPerCycle: 10}, want: &rigidController{levels: []LoadLevel{{0, 0}, {0.5, 0.5}, {1, 1}}, maxChangesPerCycle: 10}},
		{name: "5. customLoadLevel_unsorted", args: args{loadLevels: []LoadLevel{{1, 1}, {0.5, 0.5}, {0, 0}}, maxChangesPerCycle: 10}, want: &rigidController{levels: []LoadLevel{{0, 0}, {0.5, 0.5}, {1, 1}}, maxChangesPerCycle: 10}},
		{name: "6. customLoadLevel_unsorted", args: args{loadLevels: []LoadLevel{{1, 1}, {0.5, 0.5}, {0.1, 0.1}, {0.8, 0.8}, {0, 0}}, maxChangesPerCycle: 10}, want: &rigidController{levels: []LoadLevel{{0, 0}, {0.1, 0.1}, {0.5, 0.5}, {0.8, 0.8}, {1, 1}}, maxChangesPerCycle: 10}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewRigidController(tt.args.loadLevels, tt.args.maxChangesPerCycle)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRigidController() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRigidController() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_rigidController_GetDesiredWorkerNum(t *testing.T) {
	type fields struct {
		levels             LoadLevels
		maxChangesPerCycle int
	}
	type args struct {
		stats Statistics
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{name: "1. defaultLoadLevels", fields: fields{levels: defaultLoadLevels, maxChangesPerCycle: 0}, args: args{stats: Statistics{MinWorker: 5, MaxWorker: 10, BufferLength: 100, CurrWorker: 5, EnqueuedJobs: 0, FinishedJobs: 0}}, want: 5},
		{name: "2. defaultLoadLevels", fields: fields{levels: defaultLoadLevels, maxChangesPerCycle: 0}, args: args{stats: Statistics{MinWorker: 5, MaxWorker: 10, BufferLength: 100, CurrWorker: 5, EnqueuedJobs: 5, FinishedJobs: 0}}, want: 5},
		{name: "3. defaultLoadLevels", fields: fields{levels: defaultLoadLevels, maxChangesPerCycle: 4}, args: args{stats: Statistics{MinWorker: 5, MaxWorker: 10, BufferLength: 100, CurrWorker: 5, EnqueuedJobs: 10, FinishedJobs: 0}}, want: 6},
		{name: "4. defaultLoadLevels", fields: fields{levels: defaultLoadLevels, maxChangesPerCycle: 2}, args: args{stats: Statistics{MinWorker: 5, MaxWorker: 10, BufferLength: 100, CurrWorker: 5, EnqueuedJobs: 20, FinishedJobs: 0}}, want: 6},
		{name: "5. defaultLoadLevels", fields: fields{levels: defaultLoadLevels, maxChangesPerCycle: 2}, args: args{stats: Statistics{MinWorker: 5, MaxWorker: 10, BufferLength: 100, CurrWorker: 5, EnqueuedJobs: 25, FinishedJobs: 0}}, want: 7},
		{name: "6. defaultLoadLevels", fields: fields{levels: defaultLoadLevels, maxChangesPerCycle: 2}, args: args{stats: Statistics{MinWorker: 5, MaxWorker: 10, BufferLength: 100, CurrWorker: 5, EnqueuedJobs: 40, FinishedJobs: 0}}, want: 7},
		{name: "7. defaultLoadLevels", fields: fields{levels: defaultLoadLevels, maxChangesPerCycle: 2}, args: args{stats: Statistics{MinWorker: 5, MaxWorker: 10, BufferLength: 100, CurrWorker: 5, EnqueuedJobs: 50, FinishedJobs: 0}}, want: 7},
		{name: "8. defaultLoadLevels", fields: fields{levels: defaultLoadLevels, maxChangesPerCycle: 2}, args: args{stats: Statistics{MinWorker: 5, MaxWorker: 10, BufferLength: 100, CurrWorker: 5, EnqueuedJobs: 70, FinishedJobs: 0}}, want: 7},
		{name: "9. defaultLoadLevels", fields: fields{levels: defaultLoadLevels, maxChangesPerCycle: 5}, args: args{stats: Statistics{MinWorker: 5, MaxWorker: 10, BufferLength: 100, CurrWorker: 5, EnqueuedJobs: 75, FinishedJobs: 0}}, want: 10},
		{name: "10. defaultLoadLevels", fields: fields{levels: defaultLoadLevels, maxChangesPerCycle: 6}, args: args{stats: Statistics{MinWorker: 5, MaxWorker: 10, BufferLength: 100, CurrWorker: 5, EnqueuedJobs: 100, FinishedJobs: 0}}, want: 10},
		{name: "11. defaultLoadLevels", fields: fields{levels: defaultLoadLevels, maxChangesPerCycle: 10}, args: args{stats: Statistics{MinWorker: 5, MaxWorker: 10, BufferLength: 100, CurrWorker: 5, EnqueuedJobs: 120, FinishedJobs: 0}}, want: 10},
		{name: "12. customLoadLevels", fields: fields{levels: []LoadLevel{{0.1, 0.1}, {0.2, 0.2}, {0.3, 0.3}, {0.5, 0.5}, {0.7, 0.7}, {0.9, 1}}, maxChangesPerCycle: 0}, args: args{stats: Statistics{MinWorker: 1, MaxWorker: 11, BufferLength: 100, CurrWorker: 1, EnqueuedJobs: 0, FinishedJobs: 0}}, want: 1},
		{name: "13. customLoadLevels", fields: fields{levels: []LoadLevel{{0.1, 0.1}, {0.2, 0.2}, {0.3, 0.3}, {0.5, 0.5}, {0.7, 0.7}, {0.9, 1}}, maxChangesPerCycle: 0}, args: args{stats: Statistics{MinWorker: 1, MaxWorker: 11, BufferLength: 100, CurrWorker: 1, EnqueuedJobs: 10, FinishedJobs: 0}}, want: 1},
		{name: "14. customLoadLevels", fields: fields{levels: []LoadLevel{{0.1, 0.1}, {0.2, 0.2}, {0.3, 0.3}, {0.5, 0.5}, {0.7, 0.7}, {0.9, 1}}, maxChangesPerCycle: 1}, args: args{stats: Statistics{MinWorker: 1, MaxWorker: 11, BufferLength: 100, CurrWorker: 1, EnqueuedJobs: 20, FinishedJobs: 0}}, want: 2},
		{name: "15. customLoadLevels", fields: fields{levels: []LoadLevel{{0.1, 0.1}, {0.2, 0.2}, {0.3, 0.3}, {0.5, 0.5}, {0.7, 0.7}, {0.9, 1}}, maxChangesPerCycle: 3}, args: args{stats: Statistics{MinWorker: 1, MaxWorker: 11, BufferLength: 100, CurrWorker: 1, EnqueuedJobs: 30, FinishedJobs: 0}}, want: 4},
		{name: "16. customLoadLevels", fields: fields{levels: []LoadLevel{{0.1, 0.1}, {0.2, 0.2}, {0.3, 0.3}, {0.5, 0.5}, {0.7, 0.7}, {0.9, 1}}, maxChangesPerCycle: 0}, args: args{stats: Statistics{MinWorker: 1, MaxWorker: 11, BufferLength: 100, CurrWorker: 1, EnqueuedJobs: 40, FinishedJobs: 0}}, want: 1},
		{name: "17. customLoadLevels", fields: fields{levels: []LoadLevel{{0.1, 0.1}, {0.2, 0.2}, {0.3, 0.3}, {0.5, 0.5}, {0.7, 0.7}, {0.9, 1}}, maxChangesPerCycle: 4}, args: args{stats: Statistics{MinWorker: 1, MaxWorker: 11, BufferLength: 100, CurrWorker: 1, EnqueuedJobs: 50, FinishedJobs: 0}}, want: 5},
		{name: "18. customLoadLevels", fields: fields{levels: []LoadLevel{{0.1, 0.1}, {0.2, 0.2}, {0.3, 0.3}, {0.5, 0.5}, {0.7, 0.7}, {0.9, 1}}, maxChangesPerCycle: 5}, args: args{stats: Statistics{MinWorker: 1, MaxWorker: 11, BufferLength: 100, CurrWorker: 1, EnqueuedJobs: 60, FinishedJobs: 0}}, want: 6},
		{name: "19. customLoadLevels", fields: fields{levels: []LoadLevel{{0.1, 0.1}, {0.2, 0.2}, {0.3, 0.3}, {0.5, 0.5}, {0.7, 0.7}, {0.9, 1}}, maxChangesPerCycle: 8}, args: args{stats: Statistics{MinWorker: 1, MaxWorker: 11, BufferLength: 100, CurrWorker: 1, EnqueuedJobs: 70, FinishedJobs: 0}}, want: 8},
		{name: "20. customLoadLevels", fields: fields{levels: []LoadLevel{{0.1, 0.1}, {0.2, 0.2}, {0.3, 0.3}, {0.5, 0.5}, {0.7, 0.7}, {0.9, 1}}, maxChangesPerCycle: 10}, args: args{stats: Statistics{MinWorker: 1, MaxWorker: 11, BufferLength: 100, CurrWorker: 1, EnqueuedJobs: 80, FinishedJobs: 0}}, want: 8},
		{name: "21. customLoadLevels", fields: fields{levels: []LoadLevel{{0.1, 0.1}, {0.2, 0.2}, {0.3, 0.3}, {0.5, 0.5}, {0.7, 0.7}, {0.9, 1}}, maxChangesPerCycle: 10}, args: args{stats: Statistics{MinWorker: 1, MaxWorker: 11, BufferLength: 100, CurrWorker: 1, EnqueuedJobs: 90, FinishedJobs: 0}}, want: 11},
		{name: "22. customLoadLevels", fields: fields{levels: []LoadLevel{{0.1, 0.1}, {0.2, 0.2}, {0.3, 0.3}, {0.5, 0.5}, {0.7, 0.7}, {0.9, 1}}, maxChangesPerCycle: 12}, args: args{stats: Statistics{MinWorker: 1, MaxWorker: 11, BufferLength: 100, CurrWorker: 1, EnqueuedJobs: 100, FinishedJobs: 0}}, want: 11},
		{name: "23. customLoadLevels", fields: fields{levels: []LoadLevel{{0.1, 0.1}, {0.2, 0.2}, {0.3, 0.3}, {0.5, 0.5}, {0.7, 0.7}, {0.9, 1}}, maxChangesPerCycle: 0}, args: args{stats: Statistics{MinWorker: 1, MaxWorker: 11, BufferLength: 100, CurrWorker: 1, EnqueuedJobs: 200, FinishedJobs: 0}}, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &rigidController{
				levels:             tt.fields.levels,
				maxChangesPerCycle: tt.fields.maxChangesPerCycle,
			}
			if got := s.GetDesiredWorkerNum(tt.args.stats); got != tt.want {
				t.Errorf("rigidController.GetDesiredWorkerNum() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_rigidController_limitToMaxChangesPerCycle(t *testing.T) {
	type fields struct {
		levels             LoadLevels
		maxChangesPerCycle int
	}
	type args struct {
		desiredWorkerNum int
		currentWorkerNum int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{name: "1. stableWorkerNum", fields: fields{maxChangesPerCycle: 0}, args: args{desiredWorkerNum: 1, currentWorkerNum: 1}, want: 1},
		{name: "2. stableWorkerNum", fields: fields{maxChangesPerCycle: 2}, args: args{desiredWorkerNum: 1, currentWorkerNum: 1}, want: 1},
		{name: "3. expandDesired", fields: fields{maxChangesPerCycle: 2}, args: args{desiredWorkerNum: 2, currentWorkerNum: 1}, want: 2},
		{name: "4. expandLimit", fields: fields{maxChangesPerCycle: 0}, args: args{desiredWorkerNum: 4, currentWorkerNum: 1}, want: 1},
		{name: "5. expandLimit", fields: fields{maxChangesPerCycle: 2}, args: args{desiredWorkerNum: 4, currentWorkerNum: 1}, want: 3},
		{name: "6. shrinkDesired", fields: fields{maxChangesPerCycle: 2}, args: args{desiredWorkerNum: 2, currentWorkerNum: 3}, want: 2},
		{name: "7. shrinkLimit", fields: fields{maxChangesPerCycle: 2}, args: args{desiredWorkerNum: 1, currentWorkerNum: 4}, want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &rigidController{
				levels:             tt.fields.levels,
				maxChangesPerCycle: tt.fields.maxChangesPerCycle,
			}
			if got := s.limitToMaxChangesPerCycle(tt.args.desiredWorkerNum, tt.args.currentWorkerNum); got != tt.want {
				t.Errorf("rigidController.limitToMaxChangesPerCycle() = %v, want %v", got, tt.want)
			}
		})
	}
}
