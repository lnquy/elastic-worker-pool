package elastic_worker_pool

import "testing"

func Test_sensitiveController_GetDesiredWorkerNum(t *testing.T) {
	type fields struct {
		levels LoadLevels
	}
	type args struct {
		stats Statistics
	}
	// {{0.1, 0.25}, {0.25, 0.5}, {0.5, 0.75}, {0.75, 1}}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{name: "1. defaultLoadLevels", fields: fields{levels: defaultLoadLevels}, args: args{stats: Statistics{MinWorker: 5, MaxWorker: 10, BufferLength: 100, CurrWorker: 5, EnqueuedJobs: 0, FinishedJobs: 0}}, want: 5},
		{name: "2. defaultLoadLevels", fields: fields{levels: defaultLoadLevels}, args: args{stats: Statistics{MinWorker: 5, MaxWorker: 10, BufferLength: 100, CurrWorker: 5, EnqueuedJobs: 5, FinishedJobs: 0}}, want: 5},
		{name: "3. defaultLoadLevels", fields: fields{levels: defaultLoadLevels}, args: args{stats: Statistics{MinWorker: 5, MaxWorker: 10, BufferLength: 100, CurrWorker: 5, EnqueuedJobs: 10, FinishedJobs: 0}}, want: 6},
		{name: "4. defaultLoadLevels", fields: fields{levels: defaultLoadLevels}, args: args{stats: Statistics{MinWorker: 5, MaxWorker: 10, BufferLength: 100, CurrWorker: 5, EnqueuedJobs: 20, FinishedJobs: 0}}, want: 6},
		{name: "5. defaultLoadLevels", fields: fields{levels: defaultLoadLevels}, args: args{stats: Statistics{MinWorker: 5, MaxWorker: 10, BufferLength: 100, CurrWorker: 5, EnqueuedJobs: 25, FinishedJobs: 0}}, want: 7},
		{name: "6. defaultLoadLevels", fields: fields{levels: defaultLoadLevels}, args: args{stats: Statistics{MinWorker: 5, MaxWorker: 10, BufferLength: 100, CurrWorker: 5, EnqueuedJobs: 40, FinishedJobs: 0}}, want: 7},
		{name: "7. defaultLoadLevels", fields: fields{levels: defaultLoadLevels}, args: args{stats: Statistics{MinWorker: 5, MaxWorker: 10, BufferLength: 100, CurrWorker: 5, EnqueuedJobs: 50, FinishedJobs: 0}}, want: 8},
		{name: "8. defaultLoadLevels", fields: fields{levels: defaultLoadLevels}, args: args{stats: Statistics{MinWorker: 5, MaxWorker: 10, BufferLength: 100, CurrWorker: 5, EnqueuedJobs: 70, FinishedJobs: 0}}, want: 8},
		{name: "9. defaultLoadLevels", fields: fields{levels: defaultLoadLevels}, args: args{stats: Statistics{MinWorker: 5, MaxWorker: 10, BufferLength: 100, CurrWorker: 5, EnqueuedJobs: 75, FinishedJobs: 0}}, want: 10},
		{name: "10. defaultLoadLevels", fields: fields{levels: defaultLoadLevels}, args: args{stats: Statistics{MinWorker: 5, MaxWorker: 10, BufferLength: 100, CurrWorker: 5, EnqueuedJobs: 100, FinishedJobs: 0}}, want: 10},
		{name: "11. defaultLoadLevels", fields: fields{levels: defaultLoadLevels}, args: args{stats: Statistics{MinWorker: 5, MaxWorker: 10, BufferLength: 100, CurrWorker: 5, EnqueuedJobs: 120, FinishedJobs: 0}}, want: 10},
		{name: "12. customLoadLevels", fields: fields{levels: []LoadLevel{{0.1, 0.1}, {0.2, 0.2}, {0.3, 0.3}, {0.5, 0.5}, {0.7, 0.7}, {0.9, 1}}}, args: args{stats: Statistics{MinWorker: 1, MaxWorker: 11, BufferLength: 100, CurrWorker: 1, EnqueuedJobs: 0, FinishedJobs: 0}}, want: 1},
		{name: "13. customLoadLevels", fields: fields{levels: []LoadLevel{{0.1, 0.1}, {0.2, 0.2}, {0.3, 0.3}, {0.5, 0.5}, {0.7, 0.7}, {0.9, 1}}}, args: args{stats: Statistics{MinWorker: 1, MaxWorker: 11, BufferLength: 100, CurrWorker: 1, EnqueuedJobs: 10, FinishedJobs: 0}}, want: 2},
		{name: "14. customLoadLevels", fields: fields{levels: []LoadLevel{{0.1, 0.1}, {0.2, 0.2}, {0.3, 0.3}, {0.5, 0.5}, {0.7, 0.7}, {0.9, 1}}}, args: args{stats: Statistics{MinWorker: 1, MaxWorker: 11, BufferLength: 100, CurrWorker: 1, EnqueuedJobs: 20, FinishedJobs: 0}}, want: 3},
		{name: "15. customLoadLevels", fields: fields{levels: []LoadLevel{{0.1, 0.1}, {0.2, 0.2}, {0.3, 0.3}, {0.5, 0.5}, {0.7, 0.7}, {0.9, 1}}}, args: args{stats: Statistics{MinWorker: 1, MaxWorker: 11, BufferLength: 100, CurrWorker: 1, EnqueuedJobs: 30, FinishedJobs: 0}}, want: 4},
		{name: "16. customLoadLevels", fields: fields{levels: []LoadLevel{{0.1, 0.1}, {0.2, 0.2}, {0.3, 0.3}, {0.5, 0.5}, {0.7, 0.7}, {0.9, 1}}}, args: args{stats: Statistics{MinWorker: 1, MaxWorker: 11, BufferLength: 100, CurrWorker: 1, EnqueuedJobs: 40, FinishedJobs: 0}}, want: 4},
		{name: "17. customLoadLevels", fields: fields{levels: []LoadLevel{{0.1, 0.1}, {0.2, 0.2}, {0.3, 0.3}, {0.5, 0.5}, {0.7, 0.7}, {0.9, 1}}}, args: args{stats: Statistics{MinWorker: 1, MaxWorker: 11, BufferLength: 100, CurrWorker: 1, EnqueuedJobs: 50, FinishedJobs: 0}}, want: 6},
		{name: "18. customLoadLevels", fields: fields{levels: []LoadLevel{{0.1, 0.1}, {0.2, 0.2}, {0.3, 0.3}, {0.5, 0.5}, {0.7, 0.7}, {0.9, 1}}}, args: args{stats: Statistics{MinWorker: 1, MaxWorker: 11, BufferLength: 100, CurrWorker: 1, EnqueuedJobs: 60, FinishedJobs: 0}}, want: 6},
		{name: "19. customLoadLevels", fields: fields{levels: []LoadLevel{{0.1, 0.1}, {0.2, 0.2}, {0.3, 0.3}, {0.5, 0.5}, {0.7, 0.7}, {0.9, 1}}}, args: args{stats: Statistics{MinWorker: 1, MaxWorker: 11, BufferLength: 100, CurrWorker: 1, EnqueuedJobs: 70, FinishedJobs: 0}}, want: 8},
		{name: "20. customLoadLevels", fields: fields{levels: []LoadLevel{{0.1, 0.1}, {0.2, 0.2}, {0.3, 0.3}, {0.5, 0.5}, {0.7, 0.7}, {0.9, 1}}}, args: args{stats: Statistics{MinWorker: 1, MaxWorker: 11, BufferLength: 100, CurrWorker: 1, EnqueuedJobs: 80, FinishedJobs: 0}}, want: 8},
		{name: "21. customLoadLevels", fields: fields{levels: []LoadLevel{{0.1, 0.1}, {0.2, 0.2}, {0.3, 0.3}, {0.5, 0.5}, {0.7, 0.7}, {0.9, 1}}}, args: args{stats: Statistics{MinWorker: 1, MaxWorker: 11, BufferLength: 100, CurrWorker: 1, EnqueuedJobs: 90, FinishedJobs: 0}}, want: 11},
		{name: "22. customLoadLevels", fields: fields{levels: []LoadLevel{{0.1, 0.1}, {0.2, 0.2}, {0.3, 0.3}, {0.5, 0.5}, {0.7, 0.7}, {0.9, 1}}}, args: args{stats: Statistics{MinWorker: 1, MaxWorker: 11, BufferLength: 100, CurrWorker: 1, EnqueuedJobs: 100, FinishedJobs: 0}}, want: 11},
		{name: "23. customLoadLevels", fields: fields{levels: []LoadLevel{{0.1, 0.1}, {0.2, 0.2}, {0.3, 0.3}, {0.5, 0.5}, {0.7, 0.7}, {0.9, 1}}}, args: args{stats: Statistics{MinWorker: 1, MaxWorker: 11, BufferLength: 100, CurrWorker: 1, EnqueuedJobs: 200, FinishedJobs: 0}}, want: 11},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &sensitiveController{
				levels: tt.fields.levels,
			}
			if got := s.GetDesiredWorkerNum(tt.args.stats); got != tt.want {
				t.Errorf("sensitiveController.GetDesiredWorkerNum() = %v, want %v", got, tt.want)
			}
		})
	}
}
