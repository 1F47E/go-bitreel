package meta

import (
	"reflect"
	"testing"
)

func TestConvertUint64ToBytes(t *testing.T) {
	testCases := []struct {
		name string
		num  uint64
		want []byte
	}{
		{
			name: "Test 1",
			num:  1234567890,
			want: []byte{0, 0, 0, 0, 73, 150, 45, 2},
		},
		{
			name: "Test 2",
			num:  9876543210,
			want: []byte{0, 0, 0, 2, 94, 126, 210, 82},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := convertUint64ToBytes(tc.num)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
