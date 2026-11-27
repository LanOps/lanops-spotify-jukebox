package config

import "testing"

func TestValidateAuthCallbackUrl(t *testing.T) {
	cases := map[string]bool{
		"https://jukebox.example.com/auth/callback": true,
		"http://127.0.0.1:9999/auth/callback":       true,
		"http://[::1]:9999/auth/callback":           true,
		"http://localhost:9999/auth/callback":       false,
		"https://localhost:9999/auth/callback":      false,
		"http://jukebox.example.com/auth/callback":  false,
		"http://192.168.1.20:9999/auth/callback":    false,
		"ftp://127.0.0.1/auth/callback":             false,
		"not a url":                                 false,
	}
	for in, ok := range cases {
		err := ValidateAuthCallbackUrl(in)
		if ok && err != nil {
			t.Errorf("%s: unexpected error %v", in, err)
		}
		if !ok && err == nil {
			t.Errorf("%s: expected error", in)
		}
	}
}
