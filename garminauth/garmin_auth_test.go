package garminauth

import (
	"testing"
)

func TestAcquireUnauthorizedTokenAndSecret(t *testing.T) {
	stringBody, err := acquireUnauthorizedTokenAndSecret()
	if err != nil {
		t.Errorf("something went wrong! the error is %s", err.Error())
	}
	t.Log(stringBody)
}
