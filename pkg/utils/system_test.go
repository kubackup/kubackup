package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetCpuThreads(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{name: RandomString(4), want: 8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, GetCpuThreads(), "GetCpuThreads()")
		})
	}
}

func TestGetCpuCores(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{name: RandomString(4), want: 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, GetCpuCores(), "GetCpuCores()")
		})
	}
}

func TestGetSN(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: RandomString(4), want: "SDF686FDS"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, GetSN(), "GetSN()")
		})
	}
}
