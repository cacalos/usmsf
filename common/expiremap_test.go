package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type expireMapTestHelper struct {
	t      *testing.T
	expkey string
	expval interface{}
}

func TestExpireMap(t *testing.T) {
	var helper expireMapTestHelper
	helper.t = t
	expmap := NewExpireMap(0, 1*time.Second)
	expmap.Expire = helper.Expire

	assert.NotNil(t, expmap, "expmap should not be NIL.")
	//assert.Equal(t, fqdnMgr.OwnFqdn, cfg.GetString("sepp.own-fqdn"))

	v, ok := expmap.Get("test")
	assert.False(t, ok, "ok should be false.")
	assert.Nil(t, v, "value should be NIL.")
	assert.Equal(t, 0, expmap.Count(), "expmap count should be 0")

	ok = expmap.Set("test-key", "test-value", time.Now().Add(3*time.Second))
	assert.True(t, ok, "ok should be true.")

	v, ok = expmap.Get("test-key")
	assert.True(t, ok, "ok should be true.")
	assert.NotNil(t, v, "value should not be NIL.")
	assert.Equal(t, "test-value", v, "expmap result should be equal")
	assert.Equal(t, 1, expmap.Count(), "expmap count should be 1")

	v = expmap.Remove("test-key")
	assert.NotNil(t, v, "value should not be NIL.")
	assert.Equal(t, "test-value", v, "expmap result should be equal")
	assert.Equal(t, 0, expmap.Count(), "expmap count should be 0")

	t.Logf("Set data key=%v, v=%v, time=%v", "test-key", "test-value", time.Now())
	ok = expmap.Set("test-key", "test-value", time.Now().Add(3*time.Second))
	assert.True(t, ok, "ok should be true.")
	time.Sleep(2 * time.Second)
	v, ok = expmap.Get("test-key")
	assert.True(t, ok, "ok should be true.")
	assert.NotNil(t, v, "value should not be NIL.")
	assert.Equal(t, "test-value", v, "expmap result should be equal")
	assert.Equal(t, 1, expmap.Count(), "expmap count should be 1")
	time.Sleep(2 * time.Second)
	v, ok = expmap.Get("test-key")
	assert.False(t, ok, "ok should be false.")
	assert.Nil(t, v, "value should be NIL.")
	assert.Equal(t, 0, expmap.Count(), "expmap count should be 0")
	assert.Equal(t, "test-key", helper.expkey, "expire key count should be equal")
	assert.Equal(t, "test-value", helper.expval, "expire key count should be equal")

	ok = expmap.Set("test-key", "test-value", time.Now().Add(3*time.Second))
	assert.True(t, ok, "ok should be true.")
	time.Sleep(2 * time.Second)
	v, ok = expmap.GetAndUpdate("test-key", 3*time.Second)
	assert.True(t, ok, "ok should be true.")
	assert.NotNil(t, v, "value should not be NIL.")
	assert.Equal(t, "test-value", v, "expmap result should be equal")
	assert.Equal(t, 1, expmap.Count(), "expmap count should be 1")
	time.Sleep(2 * time.Second)
	v, ok = expmap.Get("test-key")
	assert.True(t, ok, "ok should be true.")
	assert.NotNil(t, v, "value should not be NIL.")
	assert.Equal(t, "test-value", v, "expmap result should be equal")
	assert.Equal(t, 1, expmap.Count(), "expmap count should be 1")

	expmap.Close()
}

func (h *expireMapTestHelper) Expire(key string, value interface{}, t *time.Time) {
	h.t.Logf("Expire data key=%v, v=%v, time=%v", key, value, t)
	h.expkey = key
	h.expval = value
}
