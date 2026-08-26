package railway

import (
	"encoding/json"
	"testing"
)

func TestBucketLifecyclePatchWireFormat(t *testing.T) {
	tests := map[string]struct {
		patch EnvironmentConfig
		want  string
	}{
		"create clears deletion": {
			patch: CreateBucketPatch("bucket-id", "ams"),
			want:  `{"buckets":{"bucket-id":{"region":"ams","isCreated":true,"isDeleted":false}}}`,
		},
		"delete clears creation": {
			patch: DeleteBucketPatch("bucket-id"),
			want:  `{"buckets":{"bucket-id":{"isCreated":false,"isDeleted":true}}}`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			payload, err := json.Marshal(test.patch)
			if err != nil {
				t.Fatalf("marshal environment patch: %v", err)
			}
			if string(payload) != test.want {
				t.Fatalf("unexpected environment patch: got %s, want %s", payload, test.want)
			}
		})
	}
}
