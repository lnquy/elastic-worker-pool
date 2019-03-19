package ewp

import "testing"

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
		{name: "2. expandDesired", fields: fields{maxChangesPerCycle: 2}, args: args{desiredWorkerNum: 2, currentWorkerNum: 1}, want: 2},
		{name: "2. expandLimit", fields: fields{maxChangesPerCycle: 2}, args: args{desiredWorkerNum: 4, currentWorkerNum: 1}, want: 3},
		{name: "2. shrinkDesired", fields: fields{maxChangesPerCycle: 2}, args: args{desiredWorkerNum: 2, currentWorkerNum: 3}, want: 2},
		{name: "2. shrinkLimit", fields: fields{maxChangesPerCycle: 2}, args: args{desiredWorkerNum: 1, currentWorkerNum: 4}, want: 2},
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
