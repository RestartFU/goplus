// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gob

import (
	"bytes"
	"reflect"
	"testing"
)

type autoRegisteredEnum enum {
	Empty
	Value { Text string }
	List { Values []int }
}

type anotherAutoRegisteredEnum enum {
	Empty
}

func TestEnumVariantsAreRegisteredAutomatically(t *testing.T) {
	tests := []struct {
		name     string
		wireName string
		value    autoRegisteredEnum
	}{
		{name: "fieldless", wireName: "encoding/gob.autoRegisteredEnum.Empty", value: autoRegisteredEnum.Empty{}},
		{name: "payload", wireName: "encoding/gob.autoRegisteredEnum.Value", value: autoRegisteredEnum.Value{Text: "hello"}},
		{name: "non-comparable payload", wireName: "encoding/gob.autoRegisteredEnum.List", value: autoRegisteredEnum.List{Values: []int{1, 2, 3}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var data bytes.Buffer
			if err := NewEncoder(&data).Encode(&test.value); err != nil {
				t.Errorf("Encode(%T) returned error: %v", test.value, err)
				return
			}
			if _, ok := nameToConcreteType.Load(test.wireName); !ok {
				t.Errorf("enum variant was not registered as %q", test.wireName)
				return
			}
			// Simulate decoding in another process, where the encoder did not
			// populate the registration maps first.
			nameToConcreteType.Delete(test.wireName)
			concreteTypeToName.Delete(reflect.TypeOf(test.value))

			var decoded autoRegisteredEnum
			if err := NewDecoder(&data).Decode(&decoded); err != nil {
				t.Errorf("Decode(%T) returned error: %v", test.value, err)
				return
			}
			if !reflect.DeepEqual(decoded, test.value) {
				t.Errorf("decoded value = %#v, want %#v", decoded, test.value)
			}
		})
	}
}

func TestEnumVariantRegistrationNamesIncludeEnumType(t *testing.T) {
	var first autoRegisteredEnum = autoRegisteredEnum.Empty{}
	var second anotherAutoRegisteredEnum = anotherAutoRegisteredEnum.Empty{}
	var data bytes.Buffer
	encoder := NewEncoder(&data)
	if err := encoder.Encode(&first); err != nil {
		t.Errorf("Encode(%T) returned error: %v", first, err)
		return
	}
	if err := encoder.Encode(&second); err != nil {
		t.Errorf("Encode(%T) returned error: %v", second, err)
	}
}
