package http

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/hashicorp/vault/helper/strutil"
	"github.com/hashicorp/vault/vault"
)

// Test to check if the API errors out when wrong number of PGP keys are
// supplied for rekey
func TestSysRekeyInit_pgpKeysEntriesForRekey(t *testing.T) {
	core, _, token := vault.TestCoreUnsealed(t)
	ln, addr := TestServer(t, core)
	defer ln.Close()
	TestServerAuth(t, addr, token)

	resp := testHttpPut(t, token, addr+"/v1/sys/rekey/init", map[string]interface{}{
		"secret_shares":    5,
		"secret_threshold": 3,
		"pgp_keys":         []string{"pgpkey1"},
	})
	testResponseStatus(t, resp, 400)
}

func TestSysRekeyInit_Status(t *testing.T) {
	core, _, token := vault.TestCoreUnsealed(t)
	ln, addr := TestServer(t, core)
	defer ln.Close()
	TestServerAuth(t, addr, token)

	resp, err := http.Get(addr + "/v1/sys/rekey/init")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	var actual map[string]interface{}
	expected := map[string]interface{}{
		"started":          false,
		"t":                json.Number("0"),
		"n":                json.Number("0"),
		"progress":         json.Number("0"),
		"required":         json.Number("3"),
		"pgp_fingerprints": interface{}(nil),
		"backup":           false,
		"nonce":            "",
	}
	testResponseStatus(t, resp, 200)
	testResponseBody(t, resp, &actual)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("\nexpected: %#v\nactual: %#v", expected, actual)
	}
}

func TestSysRekeyInit_Setup(t *testing.T) {
	core, _, token := vault.TestCoreUnsealed(t)
	ln, addr := TestServer(t, core)
	defer ln.Close()
	TestServerAuth(t, addr, token)

	resp := testHttpPut(t, token, addr+"/v1/sys/rekey/init", map[string]interface{}{
		"secret_shares":    5,
		"secret_threshold": 3,
	})
	testResponseStatus(t, resp, 200)

	var actual map[string]interface{}
	expected := map[string]interface{}{
		"started":          true,
		"t":                json.Number("3"),
		"n":                json.Number("5"),
		"progress":         json.Number("0"),
		"required":         json.Number("3"),
		"pgp_fingerprints": interface{}(nil),
		"backup":           false,
	}
	testResponseStatus(t, resp, 200)
	testResponseBody(t, resp, &actual)
	if actual["nonce"].(string) == "" {
		t.Fatalf("nonce was empty")
	}
	expected["nonce"] = actual["nonce"]
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("\nexpected: %#v\nactual: %#v", expected, actual)
	}

	resp = testHttpGet(t, token, addr+"/v1/sys/rekey/init")

	actual = map[string]interface{}{}
	expected = map[string]interface{}{
		"started":          true,
		"t":                json.Number("3"),
		"n":                json.Number("5"),
		"progress":         json.Number("0"),
		"required":         json.Number("3"),
		"pgp_fingerprints": interface{}(nil),
		"backup":           false,
	}
	testResponseStatus(t, resp, 200)
	testResponseBody(t, resp, &actual)
	if actual["nonce"].(string) == "" {
		t.Fatalf("nonce was empty")
	}
	if actual["nonce"].(string) == "" {
		t.Fatalf("nonce was empty")
	}
	expected["nonce"] = actual["nonce"]
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("\nexpected: %#v\nactual: %#v", expected, actual)
	}
}

func TestSysRekeyInit_Cancel(t *testing.T) {
	core, _, token := vault.TestCoreUnsealed(t)
	ln, addr := TestServer(t, core)
	defer ln.Close()
	TestServerAuth(t, addr, token)

	resp := testHttpPut(t, token, addr+"/v1/sys/rekey/init", map[string]interface{}{
		"secret_shares":    5,
		"secret_threshold": 3,
	})
	testResponseStatus(t, resp, 200)

	resp = testHttpDelete(t, token, addr+"/v1/sys/rekey/init")
	testResponseStatus(t, resp, 204)

	resp, err := http.Get(addr + "/v1/sys/rekey/init")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	var actual map[string]interface{}
	expected := map[string]interface{}{
		"started":          false,
		"t":                json.Number("0"),
		"n":                json.Number("0"),
		"progress":         json.Number("0"),
		"required":         json.Number("3"),
		"pgp_fingerprints": interface{}(nil),
		"backup":           false,
		"nonce":            "",
	}
	testResponseStatus(t, resp, 200)
	testResponseBody(t, resp, &actual)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("\nexpected: %#v\nactual: %#v", expected, actual)
	}
}

func TestSysRekey_badKey(t *testing.T) {
	core, _, token := vault.TestCoreUnsealed(t)
	ln, addr := TestServer(t, core)
	defer ln.Close()
	TestServerAuth(t, addr, token)

	resp := testHttpPut(t, token, addr+"/v1/sys/rekey/update", map[string]interface{}{
		"key": "0123",
	})
	testResponseStatus(t, resp, 400)
}

func TestSysRekey_SecretSharesMetadata(t *testing.T) {
	core, keys, token := vault.TestCoreUnsealed(t)
	ln, addr := TestServer(t, core)
	defer ln.Close()
	TestServerAuth(t, addr, token)

	resp := testHttpPut(t, token, addr+"/v1/sys/rekey/init", map[string]interface{}{
		"secret_shares":    5,
		"secret_threshold": 3,
	})
	var rekeyStatus map[string]interface{}
	testResponseStatus(t, resp, 200)
	testResponseBody(t, resp, &rekeyStatus)

	var actual map[string]interface{}
	for i, key := range keys {
		resp = testHttpPut(t, token, addr+"/v1/sys/rekey/update", map[string]interface{}{
			"nonce": rekeyStatus["nonce"].(string),
			"key":   hex.EncodeToString(key),
		})

		testResponseStatus(t, resp, 200)
		if i+1 == len(keys) {
			testResponseBody(t, resp, &actual)
		}
	}

	if actual == nil {
		t.Fatalf("failed to rekey")
	}

	keySharesMetadata := actual["key_shares_metadata"].([]interface{})
	if len(keySharesMetadata) != 5 {
		t.Fatalf("bad: length of key_shares_metadata: expected: 5, actual: %d", len(keySharesMetadata))
	}

	for key, item := range keySharesMetadata {
		metadata := item.(map[string]interface{})
		if metadata["id"].(string) == "" {
			t.Fatalf("bad: missing identifier in key shares metadata for key %q: %#v", key, keySharesMetadata)
		}
	}
}

func TestSysRekey_SecretSharesMetadataKeyIdentifierNames(t *testing.T) {
	core, keys, token := vault.TestCoreUnsealed(t)
	ln, addr := TestServer(t, core)
	defer ln.Close()
	TestServerAuth(t, addr, token)

	resp := testHttpPut(t, token, addr+"/v1/sys/rekey/init", map[string]interface{}{
		"secret_shares":               5,
		"secret_threshold":            3,
		"key_shares_identifier_names": "first,second,third,forth,fifth",
	})
	var rekeyStatus map[string]interface{}
	testResponseStatus(t, resp, 200)
	testResponseBody(t, resp, &rekeyStatus)

	var actual map[string]interface{}
	for i, key := range keys {
		resp = testHttpPut(t, token, addr+"/v1/sys/rekey/update", map[string]interface{}{
			"nonce": rekeyStatus["nonce"].(string),
			"key":   hex.EncodeToString(key),
		})

		testResponseStatus(t, resp, 200)
		if i+1 == len(keys) {
			testResponseBody(t, resp, &actual)
		}
	}

	if actual == nil {
		t.Fatalf("failed to rekey")
	}
	nameList := []string{"first", "second", "third", "forth", "fifth"}

	keySharesMetadata := actual["key_shares_metadata"].([]interface{})
	if len(keySharesMetadata) != 5 {
		t.Fatalf("bad: length of key_shares_metadata: expected: 5, actual: %d", len(keySharesMetadata))
	}

	for key, item := range keySharesMetadata {
		metadata := item.(map[string]interface{})
		if metadata["id"].(string) == "" {
			t.Fatalf("bad: missing identifier in key shares metadata for key %q: %#v", key, keySharesMetadata)
		}
		if metadata["name"].(string) == "" {
			t.Fatalf("invalid key identifier name")
		}
		nameList = strutil.StrListDelete(nameList, metadata["name"].(string))
	}

	if len(nameList) != 0 {
		t.Fatalf("bad: length of key identifier names list: expected 0, actual: %d", len(nameList))
	}
}

func TestSysRekey_Update(t *testing.T) {
	core, keys, token := vault.TestCoreUnsealed(t)
	ln, addr := TestServer(t, core)
	defer ln.Close()
	TestServerAuth(t, addr, token)

	resp := testHttpPut(t, token, addr+"/v1/sys/rekey/init", map[string]interface{}{
		"secret_shares":    5,
		"secret_threshold": 3,
	})
	var rekeyStatus map[string]interface{}
	testResponseStatus(t, resp, 200)
	testResponseBody(t, resp, &rekeyStatus)

	var actual map[string]interface{}
	var expected map[string]interface{}

	for i, key := range keys {
		resp = testHttpPut(t, token, addr+"/v1/sys/rekey/update", map[string]interface{}{
			"nonce": rekeyStatus["nonce"].(string),
			"key":   hex.EncodeToString(key),
		})

		actual = map[string]interface{}{}
		expected = map[string]interface{}{
			"started":          true,
			"nonce":            rekeyStatus["nonce"].(string),
			"backup":           false,
			"pgp_fingerprints": interface{}(nil),
			"required":         json.Number("3"),
			"t":                json.Number("3"),
			"n":                json.Number("5"),
			"progress":         json.Number(fmt.Sprintf("%d", i+1)),
		}
		testResponseStatus(t, resp, 200)
		testResponseBody(t, resp, &actual)

		if i+1 == len(keys) {
			delete(expected, "started")
			delete(expected, "required")
			delete(expected, "t")
			delete(expected, "n")
			delete(expected, "progress")
			expected["complete"] = true
			expected["keys"] = actual["keys"]
			expected["keys_base64"] = actual["keys_base64"]
			expected["secret_shares_metadata"] = actual["secret_shares_metadata"]
		}

		if i+1 < len(keys) && (actual["nonce"] == nil || actual["nonce"].(string) == "") {
			t.Fatalf("expected a nonce, i is %d, actual is %#v", i, actual)
		}

		if !reflect.DeepEqual(actual, expected) {
			t.Fatalf("\nexpected: \n%#v\nactual: \n%#v", expected, actual)
		}
	}

	retKeys := actual["keys"].([]interface{})
	if len(retKeys) != 5 {
		t.Fatalf("bad: %#v", retKeys)
	}
	keysB64 := actual["keys_base64"].([]interface{})
	if len(keysB64) != 5 {
		t.Fatalf("bad: %#v", keysB64)
	}
}

func TestSysRekey_ReInitUpdate(t *testing.T) {
	core, keys, token := vault.TestCoreUnsealed(t)
	ln, addr := TestServer(t, core)
	defer ln.Close()
	TestServerAuth(t, addr, token)

	resp := testHttpPut(t, token, addr+"/v1/sys/rekey/init", map[string]interface{}{
		"secret_shares":    5,
		"secret_threshold": 3,
	})
	testResponseStatus(t, resp, 200)

	resp = testHttpDelete(t, token, addr+"/v1/sys/rekey/init")
	testResponseStatus(t, resp, 204)

	resp = testHttpPut(t, token, addr+"/v1/sys/rekey/init", map[string]interface{}{
		"secret_shares":    5,
		"secret_threshold": 3,
	})
	testResponseStatus(t, resp, 200)

	resp = testHttpPut(t, token, addr+"/v1/sys/rekey/update", map[string]interface{}{
		"key": hex.EncodeToString(keys[0]),
	})

	testResponseStatus(t, resp, 400)
}
