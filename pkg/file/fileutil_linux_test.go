package fileutil

import (
	"os"
	"reflect"
	"testing"
)

func TestExist(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Exist(tt.args.name); got != tt.want {
				t.Errorf("Exist() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFixPath(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FixPath(tt.args.path); got != tt.want {
				t.Errorf("FixPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetFilePath(t *testing.T) {
	type args struct {
		file string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetFilePath(tt.args.file); got != tt.want {
				t.Errorf("GetFilePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHomeDir(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HomeDir(); got != tt.want {
				t.Errorf("HomeDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListDir(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    []*model.FileInfo
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ListDir(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ListDir() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMkdir(t *testing.T) {
	type args struct {
		path string
		mode os.FileMode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Mkdir(tt.args.path, tt.args.mode); got != tt.want {
				t.Errorf("Mkdir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReplaceHomeDir(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ReplaceHomeDir(tt.args.path); got != tt.want {
				t.Errorf("ReplaceHomeDir() = %v, want %v", got, tt.want)
			}
		})
	}
}
