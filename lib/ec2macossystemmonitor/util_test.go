package ec2macossystemmonitor

import (
	"runtime"
	"testing"
)

// Test_fileExists tests the fileExists helper function with basic coverage
func Test_fileExists(t *testing.T) {
	_, testFile, _, _ := runtime.Caller(0)
	type args struct {
		filename string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"Not Real File", args{"notafile"}, false},
		{"Known File", args{testFile}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fileExists(tt.args.filename); got != tt.want {
				t.Errorf("fileExists() = %v, want %v", got, tt.want)
			}
		})
	}
}
