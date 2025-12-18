package domain

import (
	"testing"
)

func TestMakeFileName(t *testing.T) {
	type args struct {
		dir  string
		name string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "1",
			args: args{
				dir:  "../data",
				name: "file",
			},
			want: "../data/file",
		},
		{
			name: "2",
			args: args{
				dir:  "../data/",
				name: "file",
			},
			want: "../data/file",
		},
		{
			name: "3",
			args: args{
				dir:  "../data/",
				name: "/file",
			},
			want: "../data/file",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MakeFileName(tt.args.dir, tt.args.name); got != tt.want {
				t.Errorf("MakeFileName() = %v, want %v", got, tt.want)
			}
		})
	}
}
