// (c) 2021 Opendoor Labs Inc.
// This code is licenced under the MIT licence (see the LICENCE file in the repo root).
package config

type Config struct {
	HomepageIsPublic bool `json:"homepage_is_public"`
	Deactivated      bool `json:"deactivated"`
}
