package models

import (
	"reflect"
	"unsafe"
)

// expose some private stuff for testing purposes

func (p *PGClient) ClearCaches() {
	// It would be nice if we could take advantage of zero values by just clobbering the
	// PGClient with:
	//
	// ```
	// newClient := PGClient{impl: p.impl, topLevelDB: p.topLevelDB}
	// *p = newClient
	// ```
	//
	// But then we will be copying/clobbering mutexes which govet is not a fan of.

	v := reflect.ValueOf(p).Elem()
	emptySlice := []int{}
	emptySliceV := reflect.ValueOf(emptySlice)
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if field.Type().Kind() != reflect.Slice {
			continue
		}

		// We resort to unsafe shenannigans to enable us to use reflection to set
		// unexported fields. We are not really breaking privacy here because the
		// PGClient type is defined in this module.
		field = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
		field.Set(emptySliceV)
	}
}

func (tx *TxPGClient) ClearCaches() {
	tx.impl.client.ClearCaches()
}
