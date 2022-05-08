package parse_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/mtlynch/picoshare/v2/handlers/parse"
	"github.com/mtlynch/picoshare/v2/types"
)

func TestExpirationDate(t *testing.T) {
	for _, tt := range []struct {
		description string
		input       string
		output      types.ExpirationTime
		err         error
	}{
		{
			description: "valid expiration",
			input:       "2025-01-01",
			output:      mustParseExpiration("2025-01-01"),
			err:         nil,
		},
		{
			description: "empty string is equivalent to no expiration",
			input:       "",
			output:      types.NeverExpire,
			err:         nil,
		},
		{
			description: "string with letters causes error",
			input:       "banana",
			output:      types.ExpirationTime{},
			err:         nil, // TODO: Expect a better error.
		},
	} {
		t.Run(fmt.Sprintf("%s [%s]", tt.description, tt.input), func(t *testing.T) {
			ed, err := parse.ExpirationDate(tt.input)
			if got, want := err, tt.err; got != want {
				t.Fatalf("err=%v, want=%v", reflect.TypeOf(err), want)
			}
			if got, want := ed, tt.output; got != want {
				t.Errorf("filename=%v, want=%v", got, want)
			}
		})
	}
}

func mustParseExpiration(s string) types.ExpirationTime {
	et, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return types.ExpirationTime(et)
}
