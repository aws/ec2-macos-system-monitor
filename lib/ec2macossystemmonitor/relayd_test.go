package ec2macossystemmonitor

import (
	"reflect"
	"testing"
)

// TestBuildMessage creates some basic tests to ensure the options result in the correct bytes
func TestBuildMessage(t *testing.T) {
	// emptyTestBytes is the byte slice of a payload tag "test" and empty payload
	emptyTestBytes := []byte{123, 34, 99, 115, 117, 109, 34, 58, 53, 48, 55, 53, 55, 57, 55, 52, 52, 44, 34, 112, 97, 121, 108, 111, 97, 100, 34, 58, 34, 123, 92, 34, 116, 97, 103, 92, 34, 58, 92, 34, 116, 101, 115, 116, 92, 34, 44, 92, 34, 99, 111, 109, 112, 114, 101, 115, 115, 92, 34, 58, 102, 97, 108, 115, 101, 44, 92, 34, 100, 97, 116, 97, 92, 34, 58, 92, 34, 92, 34, 125, 34, 125, 10}
	// emptyTestBytes is the byte slice of a payload tag "test" and empty payload while compressing ""
	emptyCompressedTestBytes := []byte{123, 34, 99, 115, 117, 109, 34, 58, 49, 54, 57, 53, 52, 54, 49, 51, 56, 44, 34, 112, 97, 121, 108, 111, 97, 100, 34, 58, 34, 123, 92, 34, 116, 97, 103, 92, 34, 58, 92, 34, 116, 101, 115, 116, 92, 34, 44, 92, 34, 99, 111, 109, 112, 114, 101, 115, 115, 92, 34, 58, 116, 114, 117, 101, 44, 92, 34, 100, 97, 116, 97, 92, 34, 58, 92, 34, 101, 78, 111, 66, 65, 65, 68, 47, 47, 119, 65, 65, 65, 65, 69, 61, 92, 34, 125, 34, 125, 10}
	basicCPUTestBytes := []byte{123, 34, 99, 115, 117, 109, 34, 58, 50, 49, 49, 56, 49, 57, 50, 57, 53, 48, 44, 34, 112, 97, 121, 108, 111, 97, 100, 34, 58, 34, 123, 92, 34, 116, 97, 103, 92, 34, 58, 92, 34, 99, 112, 117, 117, 116, 105, 108, 92, 34, 44, 92, 34, 99, 111, 109, 112, 114, 101, 115, 115, 92, 34, 58, 102, 97, 108, 115, 101, 44, 92, 34, 100, 97, 116, 97, 92, 34, 58, 92, 34, 50, 46, 48, 92, 34, 125, 34, 125, 10}

	type args struct {
		tag      string
		data     string
		compress bool
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{"Empty Message", args{"test", "", false}, emptyTestBytes, false},
		{"Empty Message Compressed", args{"test", "", true}, emptyCompressedTestBytes, false},
		{"Basic CPU Test", args{"cpuutil", "2.0", false}, basicCPUTestBytes, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildMessage(tt.args.tag, tt.args.data, tt.args.compress)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildMessage() got = %v, want %v", got, tt.want)
			}
		})
	}
}
