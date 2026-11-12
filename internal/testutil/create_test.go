package testutil_test

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/mtlynch/picoshare/internal/testutil"
)

type fakeTB struct {
	testing.TB
	helperCalled bool
	fatalMessage string
}

func (t *fakeTB) Helper() {
	t.helperCalled = true
}

func (t *fakeTB) Fatalf(format string, args ...any) {
	t.fatalMessage = fmt.Sprintf(format, args...)
}

func TestMustCreate(t *testing.T) {
	t.Run("constructor succeeds", func(t *testing.T) {
		value := testutil.MustCreate(t, strconv.Atoi, "42")
		if got, want := value, 42; got != want {
			t.Errorf("value=%d, want=%d", got, want)
		}
	})

	t.Run("constructor fails", func(t *testing.T) {
		fake := &fakeTB{}
		wantErr := errors.New("constructor error")
		testutil.MustCreate(fake, func(input int) (string, error) {
			return "", wantErr
		}, 1)
		if got, want := fake.helperCalled, true; got != want {
			t.Errorf("Helper called=%t, want=%t", got, want)
		}
		if got, want := fake.fatalMessage, "failed to create test value: constructor error"; got != want {
			t.Errorf("Fatalf message=%q, want=%q", got, want)
		}
	})
}

func TestMustCreate2(t *testing.T) {
	t.Run("constructor succeeds", func(t *testing.T) {
		value := testutil.MustCreate2(t, func(label string, number int) (string, error) {
			return fmt.Sprintf("%s-%d", label, number), nil
		}, "file", 42)
		if got, want := value, "file-42"; got != want {
			t.Errorf("value=%q, want=%q", got, want)
		}
	})

	t.Run("constructor fails", func(t *testing.T) {
		fake := &fakeTB{}
		wantErr := errors.New("constructor error")
		testutil.MustCreate2(fake, func(first, second int) (int, error) {
			return 0, wantErr
		}, 2, 3)
		if got, want := fake.helperCalled, true; got != want {
			t.Errorf("Helper called=%t, want=%t", got, want)
		}
		if got, want := fake.fatalMessage, "failed to create test value: constructor error"; got != want {
			t.Errorf("Fatalf message=%q, want=%q", got, want)
		}
	})
}
