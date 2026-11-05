package shim

import "github.com/ejoja/shortcuts/internal/config"

func validateShortcutName(name string) error {
	return config.ValidateName(name)
}
