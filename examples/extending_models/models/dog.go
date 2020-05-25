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
