// (c) 2021 Opendoor Labs Inc.
// This code is licenced under the MIT licence (see the LICENCE file in the repo root).
package models

// extensions for the dog struct

func (d *Dog) Bark() string {
	switch d.Size {
	case SizeCategorySmall:
		return "yip"
	case SizeCategoryLarge:
		return "woof"
	default:
		return "unknown dog category"
	}
}
