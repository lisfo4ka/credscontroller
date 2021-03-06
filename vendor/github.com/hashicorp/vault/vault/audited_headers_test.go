package vault

import (
	"reflect"
	"testing"

	"github.com/hashicorp/vault/helper/salt"
)

func mockAuditedHeadersConfig(t *testing.T) *AuditedHeadersConfig {
	_, barrier, _ := mockBarrier(t)
	view := NewBarrierView(barrier, "foo/")
	return &AuditedHeadersConfig{
		Headers: make(map[string]*auditedHeaderSettings),
		view:    view,
	}
}

func TestAuditedHeadersConfig_CRUD(t *testing.T) {
	conf := mockAuditedHeadersConfig(t)

	testAuditedHeadersConfig_Add(t, conf)
	testAuditedHeadersConfig_Remove(t, conf)
}

func testAuditedHeadersConfig_Add(t *testing.T, conf *AuditedHeadersConfig) {
	err := conf.add("X-Test-Header", false)
	if err != nil {
		t.Fatalf("Error when adding header to config: %s", err)
	}

	settings, ok := conf.Headers["X-Test-Header"]
	if !ok {
		t.Fatal("Expected header to be found in config")
	}

	if settings.HMAC {
		t.Fatal("Expected HMAC to be set to false, got true")
	}

	out, err := conf.view.Get(auditedHeadersEntry)
	if err != nil {
		t.Fatalf("Could not retrieve headers entry from config: %s", err)
	}

	headers := make(map[string]*auditedHeaderSettings)
	err = out.DecodeJSON(&headers)
	if err != nil {
		t.Fatalf("Error decoding header view: %s", err)
	}

	expected := map[string]*auditedHeaderSettings{
		"X-Test-Header": &auditedHeaderSettings{
			HMAC: false,
		},
	}

	if !reflect.DeepEqual(headers, expected) {
		t.Fatalf("Expected config didn't match actual. Expected: %#v, Got: %#v", expected, headers)
	}

	err = conf.add("X-Vault-Header", true)
	if err != nil {
		t.Fatalf("Error when adding header to config: %s", err)
	}

	settings, ok = conf.Headers["X-Vault-Header"]
	if !ok {
		t.Fatal("Expected header to be found in config")
	}

	if !settings.HMAC {
		t.Fatal("Expected HMAC to be set to true, got false")
	}

	out, err = conf.view.Get(auditedHeadersEntry)
	if err != nil {
		t.Fatalf("Could not retrieve headers entry from config: %s", err)
	}

	headers = make(map[string]*auditedHeaderSettings)
	err = out.DecodeJSON(&headers)
	if err != nil {
		t.Fatalf("Error decoding header view: %s", err)
	}

	expected["X-Vault-Header"] = &auditedHeaderSettings{
		HMAC: true,
	}

	if !reflect.DeepEqual(headers, expected) {
		t.Fatalf("Expected config didn't match actual. Expected: %#v, Got: %#v", expected, headers)
	}

}

func testAuditedHeadersConfig_Remove(t *testing.T, conf *AuditedHeadersConfig) {
	err := conf.remove("X-Test-Header")
	if err != nil {
		t.Fatalf("Error when adding header to config: %s", err)
	}

	_, ok := conf.Headers["X-Test-Header"]
	if ok {
		t.Fatal("Expected header to not be found in config")
	}

	out, err := conf.view.Get(auditedHeadersEntry)
	if err != nil {
		t.Fatalf("Could not retrieve headers entry from config: %s", err)
	}

	headers := make(map[string]*auditedHeaderSettings)
	err = out.DecodeJSON(&headers)
	if err != nil {
		t.Fatalf("Error decoding header view: %s", err)
	}

	expected := map[string]*auditedHeaderSettings{
		"X-Vault-Header": &auditedHeaderSettings{
			HMAC: true,
		},
	}

	if !reflect.DeepEqual(headers, expected) {
		t.Fatalf("Expected config didn't match actual. Expected: %#v, Got: %#v", expected, headers)
	}

	err = conf.remove("X-Vault-Header")
	if err != nil {
		t.Fatalf("Error when adding header to config: %s", err)
	}

	_, ok = conf.Headers["X-Vault-Header"]
	if ok {
		t.Fatal("Expected header to not be found in config")
	}

	out, err = conf.view.Get(auditedHeadersEntry)
	if err != nil {
		t.Fatalf("Could not retrieve headers entry from config: %s", err)
	}

	headers = make(map[string]*auditedHeaderSettings)
	err = out.DecodeJSON(&headers)
	if err != nil {
		t.Fatalf("Error decoding header view: %s", err)
	}

	expected = make(map[string]*auditedHeaderSettings)

	if !reflect.DeepEqual(headers, expected) {
		t.Fatalf("Expected config didn't match actual. Expected: %#v, Got: %#v", expected, headers)
	}
}

func TestAuditedHeadersConfig_ApplyConfig(t *testing.T) {
	conf := mockAuditedHeadersConfig(t)

	conf.Headers = map[string]*auditedHeaderSettings{
		"X-Test-Header":  &auditedHeaderSettings{false},
		"X-Vault-Header": &auditedHeaderSettings{true},
	}

	reqHeaders := map[string][]string{
		"X-Test-Header":  []string{"foo"},
		"X-Vault-Header": []string{"bar", "bar"},
		"Content-Type":   []string{"json"},
	}

	hashFunc := func(s string) string { return "hashed" }

	result := conf.ApplyConfig(reqHeaders, hashFunc)

	expected := map[string][]string{
		"X-Test-Header":  []string{"foo"},
		"X-Vault-Header": []string{"hashed", "hashed"},
	}

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("Expected headers did not match actual: Expected %#v\n Got %#v\n", expected, result)
	}

	//Make sure we didn't edit the reqHeaders map
	reqHeadersCopy := map[string][]string{
		"X-Test-Header":  []string{"foo"},
		"X-Vault-Header": []string{"bar", "bar"},
		"Content-Type":   []string{"json"},
	}

	if !reflect.DeepEqual(reqHeaders, reqHeadersCopy) {
		t.Fatalf("Req headers were changed, expected %#v\n got %#v", reqHeadersCopy, reqHeaders)
	}

}

func BenchmarkAuditedHeaderConfig_ApplyConfig(b *testing.B) {
	conf := &AuditedHeadersConfig{
		Headers: make(map[string]*auditedHeaderSettings),
		view:    nil,
	}

	conf.Headers = map[string]*auditedHeaderSettings{
		"X-Test-Header":  &auditedHeaderSettings{false},
		"X-Vault-Header": &auditedHeaderSettings{true},
	}

	reqHeaders := map[string][]string{
		"X-Test-Header":  []string{"foo"},
		"X-Vault-Header": []string{"bar", "bar"},
		"Content-Type":   []string{"json"},
	}

	salter, err := salt.NewSalt(nil, nil)
	if err != nil {
		b.Fatal(err)
	}

	hashFunc := func(s string) string { return salter.GetIdentifiedHMAC(s) }

	// Reset the timer since we did a lot above
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conf.ApplyConfig(reqHeaders, hashFunc)
	}
}
